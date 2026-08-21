package localqueue

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/coffeehc/base/log"
	"go.uber.org/zap"
)

const (
	metadataFileSuffix  = ".diskqueue.meta.dat"
	segmentFileSuffix   = ".dat"
	badFileSuffix       = ".bad"
	temporaryFileSuffix = ".tmp"
)

// queueMetadata 是兼容 go-diskqueue 文本格式的持久化 checkpoint。
type queueMetadata struct {
	// depth 是 checkpoint 时尚未消费的消息数。
	depth int64
	// readFileNum 是 checkpoint 时的读取 segment 编号。
	readFileNum int64
	// readPosition 是 checkpoint 时的读取偏移。
	readPosition int64
	// writeFileNum 是 checkpoint 时的写入 segment 编号。
	writeFileNum int64
	// writePosition 是 checkpoint 时的写入偏移。
	writePosition int64
}

func (queue *diskQueue) initializeStorage() error {
	if err := queue.validateStorageConfig(); err != nil {
		return err
	}
	absoluteQueueDir, err := filepath.Abs(queue.queueDir)
	if err != nil {
		return fmt.Errorf("解析队列目录绝对路径失败: %w", err)
	}
	queue.queueDir = absoluteQueueDir
	if err := os.MkdirAll(queue.queueDir, 0o700); err != nil {
		return fmt.Errorf("创建队列目录失败: %w", err)
	}
	resolvedQueueDir, err := filepath.EvalSymlinks(queue.queueDir)
	if err != nil {
		return fmt.Errorf("解析队列目录真实路径失败: %w", err)
	}
	queue.queueDir = resolvedQueueDir
	queue.ownership, err = acquireQueueOwnership(queue.name, queue.queueDir)
	if err != nil {
		return err
	}
	metadataLoaded, err := queue.loadMetadata()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Warn("读取本地磁盘队列元数据失败，将从 segment 恢复",
			zap.String("queue", queue.name),
			zap.Error(err),
		)
		metadataLoaded = false
	}
	changed, err := queue.reconcileStorage(metadataLoaded)
	if err != nil {
		return err
	}
	if changed {
		queue.dirty = true
		if err = queue.syncStorage(); err != nil {
			return fmt.Errorf("保存恢复后的队列元数据失败: %w", err)
		}
	}
	return nil
}

func (queue *diskQueue) validateStorageConfig() error {
	if queue.name == "" || queue.name == "." || queue.name == ".." ||
		filepath.Base(queue.name) != queue.name || strings.ContainsAny(queue.name, `/\`) {
		return fmt.Errorf("队列名称 %q 不是安全文件名", queue.name)
	}
	if queue.queueDir == "" {
		return errors.New("队列目录不能为空")
	}
	if queue.config.maxBytesPerFile <= 0 || queue.config.minMessageSize < 0 ||
		queue.config.maxMessageSize < queue.config.minMessageSize || queue.config.syncEvery <= 0 ||
		queue.config.syncTimeout <= 0 {
		return errors.New("队列配置非法")
	}
	return nil
}

func (queue *diskQueue) loadMetadata() (bool, error) {
	file, err := os.Open(queue.metadataPath())
	if err != nil {
		return false, err
	}
	defer file.Close()
	metadata := &queueMetadata{}
	if _, err = fmt.Fscanf(file, "%d\n%d,%d\n%d,%d\n",
		&metadata.depth,
		&metadata.readFileNum,
		&metadata.readPosition,
		&metadata.writeFileNum,
		&metadata.writePosition,
	); err != nil {
		return false, fmt.Errorf("解析队列元数据失败: %w", err)
	}
	queue.depth.Store(metadata.depth)
	queue.readFileNum = metadata.readFileNum
	queue.readPosition = metadata.readPosition
	queue.writeFileNum = metadata.writeFileNum
	queue.writePosition = metadata.writePosition
	return true, nil
}

// reconcileStorage 在正常启动时只扫描 checkpoint 之后的尾部，异常状态才全量重建。
func (queue *diskQueue) reconcileStorage(metadataLoaded bool) (bool, error) {
	segments, err := queue.segmentNumbers()
	if err != nil {
		return false, err
	}
	if !metadataLoaded || !queue.metadataStateValid(segments) {
		if metadataLoaded {
			log.Warn("本地磁盘队列元数据与 segment 不一致，将全量恢复",
				zap.String("queue", queue.name),
			)
		}
		if err = queue.rebuildFromSegments(segments); err != nil {
			return false, err
		}
		return len(segments) > 0 || metadataLoaded, nil
	}

	changed := false
	for _, fileNum := range segments {
		if fileNum < queue.writeFileNum {
			continue
		}
		startPosition := int64(0)
		if fileNum == queue.writeFileNum {
			startPosition = queue.writePosition
		}
		endPosition, messageCount, repaired, scanErr := queue.scanSegment(fileNum, startPosition)
		if scanErr != nil {
			return false, scanErr
		}
		if fileNum > queue.writeFileNum || endPosition != queue.writePosition || messageCount > 0 || repaired {
			changed = true
		}
		queue.writeFileNum = fileNum
		queue.writePosition = endPosition
		if messageCount > 0 {
			queue.depth.Add(messageCount)
		}
	}
	return changed, nil
}

func (queue *diskQueue) metadataStateValid(segments []int64) bool {
	depth := queue.depth.Load()
	if depth < 0 || queue.readFileNum < 0 || queue.writeFileNum < 0 ||
		queue.readPosition < 0 || queue.writePosition < 0 ||
		queue.readFileNum > queue.writeFileNum {
		return false
	}
	if len(segments) == 0 {
		return depth == 0 && queue.readPosition == 0 && queue.writePosition == 0
	}
	if depth > 0 && !slices.Contains(segments, queue.readFileNum) {
		return false
	}
	if queue.writePosition > 0 && !slices.Contains(segments, queue.writeFileNum) {
		return false
	}
	if slices.Contains(segments, queue.readFileNum) {
		info, err := os.Stat(queue.segmentPath(queue.readFileNum))
		if err != nil || queue.readPosition > info.Size() {
			return false
		}
	}
	if slices.Contains(segments, queue.writeFileNum) {
		info, err := os.Stat(queue.segmentPath(queue.writeFileNum))
		if err != nil || queue.writePosition > info.Size() {
			return false
		}
	}
	return true
}

// rebuildFromSegments 将现存 segment 中的全部完整消息作为未消费消息恢复。
func (queue *diskQueue) rebuildFromSegments(segments []int64) error {
	queue.depth.Store(0)
	queue.readFileNum = 0
	queue.readPosition = 0
	queue.writeFileNum = 0
	queue.writePosition = 0
	firstReadable := int64(-1)
	for _, fileNum := range segments {
		endPosition, messageCount, repaired, err := queue.scanSegment(fileNum, 0)
		if err != nil {
			return err
		}
		if repaired {
			log.Warn("修复本地磁盘队列 segment 尾部",
				zap.String("queue", queue.name),
				zap.Int64("segment", fileNum),
			)
		}
		if firstReadable < 0 && messageCount > 0 {
			firstReadable = fileNum
		}
		queue.depth.Add(messageCount)
		queue.writeFileNum = fileNum
		queue.writePosition = endPosition
	}
	if firstReadable >= 0 {
		queue.readFileNum = firstReadable
	} else {
		queue.readFileNum = queue.writeFileNum
		queue.readPosition = queue.writePosition
	}
	return nil
}

// scanSegment 校验从 startPosition 开始的消息帧，并截断崩溃留下的不完整尾部。
func (queue *diskQueue) scanSegment(fileNum, startPosition int64) (int64, int64, bool, error) {
	file, err := os.OpenFile(queue.segmentPath(fileNum), os.O_RDWR, 0o600)
	if err != nil {
		return 0, 0, false, fmt.Errorf("打开队列 segment %d 失败: %w", fileNum, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, 0, false, err
	}
	if startPosition < 0 || startPosition > info.Size() {
		return 0, 0, false, fmt.Errorf("segment %d 起始位置 %d 超出文件大小 %d", fileNum, startPosition, info.Size())
	}
	position := startPosition
	messageCount := int64(0)
	header := make([]byte, 4)
	for position < info.Size() {
		if info.Size()-position < 4 {
			if err = file.Truncate(position); err != nil {
				return 0, 0, false, err
			}
			return position, messageCount, true, file.Sync()
		}
		if _, err = file.ReadAt(header, position); err != nil {
			return 0, 0, false, err
		}
		messageSize := int32(binary.BigEndian.Uint32(header))
		if messageSize < queue.config.minMessageSize || messageSize > queue.config.maxMessageSize ||
			position+4+int64(messageSize) > info.Size() {
			if err = file.Truncate(position); err != nil {
				return 0, 0, false, err
			}
			return position, messageCount, true, file.Sync()
		}
		position += 4 + int64(messageSize)
		messageCount++
	}
	return position, messageCount, false, nil
}

func (queue *diskQueue) writeMessage(message []byte) error {
	if err := queue.validateMessage(message); err != nil {
		return err
	}
	frameSize := int64(4 + len(message))
	if queue.writePosition > 0 && queue.writePosition+frameSize > queue.config.maxBytesPerFile {
		if err := queue.syncStorage(); err != nil {
			return err
		}
		if err := queue.closeWriteFile(); err != nil {
			return err
		}
		queue.writeFileNum++
		queue.writePosition = 0
	}
	if err := queue.openWriteFile(); err != nil {
		return err
	}
	frameLength := int(frameSize)
	queue.writeBuffer = slices.Grow(queue.writeBuffer[:0], frameLength)[:frameLength]
	binary.BigEndian.PutUint32(queue.writeBuffer[:4], uint32(len(message)))
	copy(queue.writeBuffer[4:], message)
	startPosition := queue.writePosition
	written, err := queue.writeFile.Write(queue.writeBuffer)
	if err == nil && written != frameLength {
		err = io.ErrShortWrite
	}
	if err != nil {
		_ = queue.writeFile.Truncate(startPosition)
		_, _ = queue.writeFile.Seek(startPosition, io.SeekStart)
		return fmt.Errorf("写入队列消息失败: %w", err)
	}
	queue.writePosition += frameSize
	queue.depth.Add(1)
	queue.dirty = true
	queue.operationCount++
	return nil
}

func (queue *diskQueue) readNextMessage() (*pendingMessage, error) {
	for queue.depth.Load() > 0 {
		if err := queue.openReadFile(); err != nil {
			return nil, err
		}
		info, err := queue.readFile.Stat()
		if err != nil {
			return nil, err
		}
		if queue.readPosition >= info.Size() && queue.readFileNum < queue.writeFileNum {
			nextFileNum, exists := queue.nextSegmentNumber(queue.readFileNum)
			if !exists {
				return nil, fmt.Errorf("segment %d 之后缺少待读 segment", queue.readFileNum)
			}
			queue.advanceReadFile(nextFileNum)
			continue
		}
		header := make([]byte, 4)
		if _, err = queue.readFile.ReadAt(header, queue.readPosition); err != nil {
			return nil, err
		}
		messageSize := int32(binary.BigEndian.Uint32(header))
		if messageSize < queue.config.minMessageSize || messageSize > queue.config.maxMessageSize {
			return nil, fmt.Errorf("读取到非法消息大小 %d", messageSize)
		}
		payload := make([]byte, messageSize)
		if _, err = queue.readFile.ReadAt(payload, queue.readPosition+4); err != nil {
			return nil, err
		}
		nextFileNum := queue.readFileNum
		nextPosition := queue.readPosition + 4 + int64(messageSize)
		if queue.readFileNum < queue.writeFileNum && nextPosition >= info.Size() {
			var exists bool
			nextFileNum, exists = queue.nextSegmentNumber(queue.readFileNum)
			if !exists {
				return nil, fmt.Errorf("segment %d 之后缺少下一 segment", queue.readFileNum)
			}
			nextPosition = 0
		}
		return &pendingMessage{payload: payload, nextFileNum: nextFileNum, nextPosition: nextPosition}, nil
	}
	return nil, nil
}

func (queue *diskQueue) commitRead(message *pendingMessage) {
	oldFileNum := queue.readFileNum
	queue.readFileNum = message.nextFileNum
	queue.readPosition = message.nextPosition
	remaining := queue.depth.Add(-1)
	if oldFileNum != queue.readFileNum {
		_ = queue.closeReadFile()
		if err := os.Remove(queue.segmentPath(oldFileNum)); err != nil && !errors.Is(err, os.ErrNotExist) {
			log.Error("删除已经消费的队列 segment 失败",
				zap.String("queue", queue.name),
				zap.Int64("segment", oldFileNum),
				zap.Error(err),
			)
		}
	}
	if remaining == 0 {
		queue.readFileNum = queue.writeFileNum
		queue.readPosition = queue.writePosition
	}
	queue.dirty = true
	queue.operationCount++
}

func (queue *diskQueue) advanceReadFile(nextFileNum int64) {
	oldFileNum := queue.readFileNum
	_ = queue.closeReadFile()
	queue.readFileNum = nextFileNum
	queue.readPosition = 0
	queue.dirty = true
	if err := os.Remove(queue.segmentPath(oldFileNum)); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("删除空队列 segment 失败", zap.String("queue", queue.name), zap.Error(err))
	}
}

func (queue *diskQueue) handleReadFailure(readErr error) {
	corruptFileNum := queue.readFileNum
	log.Error("读取本地磁盘队列失败，将隔离损坏 segment",
		zap.String("queue", queue.name),
		zap.Int64("segment", corruptFileNum),
		zap.Error(readErr),
	)
	_ = queue.closeReadFile()
	if corruptFileNum == queue.writeFileNum {
		_ = queue.closeWriteFile()
	}
	corruptPath := queue.segmentPath(corruptFileNum)
	badPath := corruptPath + ".bad"
	if _, err := os.Stat(badPath); err == nil {
		badPath = fmt.Sprintf("%s.%d", badPath, time.Now().UnixNano())
	}
	if err := os.Rename(corruptPath, badPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Error("隔离损坏队列 segment 失败", zap.String("queue", queue.name), zap.Error(err))
	}
	segments, err := queue.segmentNumbers()
	if err == nil {
		err = queue.rebuildFromSegments(segments)
	}
	if err != nil {
		queue.depth.Store(0)
		queue.writeFileNum = max(queue.writeFileNum, corruptFileNum) + 1
		queue.writePosition = 0
		queue.readFileNum = queue.writeFileNum
		queue.readPosition = 0
		log.Error("恢复损坏队列失败，已移动到新的空 segment",
			zap.String("queue", queue.name),
			zap.Error(err),
		)
	}
	if queue.writeFileNum <= corruptFileNum {
		queue.writeFileNum = corruptFileNum + 1
		queue.writePosition = 0
		if queue.depth.Load() == 0 {
			queue.readFileNum = queue.writeFileNum
			queue.readPosition = 0
		}
	}
	queue.dirty = true
}

func (queue *diskQueue) openReadFile() error {
	if queue.readFile != nil {
		return nil
	}
	file, err := os.Open(queue.segmentPath(queue.readFileNum))
	if err != nil {
		return fmt.Errorf("打开读取 segment %d 失败: %w", queue.readFileNum, err)
	}
	queue.readFile = file
	return nil
}

func (queue *diskQueue) openWriteFile() error {
	if queue.writeFile != nil {
		return nil
	}
	file, err := os.OpenFile(queue.segmentPath(queue.writeFileNum), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("打开写入 segment %d 失败: %w", queue.writeFileNum, err)
	}
	if _, err = file.Seek(queue.writePosition, io.SeekStart); err != nil {
		file.Close()
		return err
	}
	queue.writeFile = file
	return nil
}

func (queue *diskQueue) syncStorage() error {
	if queue.writeFile != nil {
		if err := queue.writeFile.Sync(); err != nil {
			return err
		}
	}
	if err := queue.persistMetadata(); err != nil {
		return err
	}
	queue.dirty = false
	queue.operationCount = 0
	return nil
}

func (queue *diskQueue) persistMetadata() error {
	temporary, err := os.CreateTemp(queue.queueDir, "."+queue.name+".diskqueue.meta.*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err = temporary.Chmod(0o600); err == nil {
		_, err = fmt.Fprintf(temporary, "%d\n%d,%d\n%d,%d\n",
			queue.depth.Load(),
			queue.readFileNum,
			queue.readPosition,
			queue.writeFileNum,
			queue.writePosition,
		)
	}
	if err == nil {
		err = temporary.Sync()
	}
	closeErr := temporary.Close()
	if err != nil || closeErr != nil {
		return errors.Join(err, closeErr)
	}
	if err = os.Rename(temporaryPath, queue.metadataPath()); err != nil {
		return err
	}
	removeTemporary = false
	return syncDirectory(queue.queueDir)
}

func (queue *diskQueue) emptyStorage() error {
	closeErr := queue.closeStorageFiles()
	removeErr := queue.removeQueueFiles()
	nextFileNum := queue.writeFileNum + 1
	queue.readFileNum = nextFileNum
	queue.readPosition = 0
	queue.writeFileNum = nextFileNum
	queue.writePosition = 0
	queue.depth.Store(0)
	queue.dirty = false
	queue.operationCount = 0
	return errors.Join(closeErr, removeErr)
}

func (queue *diskQueue) shutdownStorage(deleteQueue bool) error {
	if deleteQueue {
		return errors.Join(queue.closeStorageFiles(), queue.removeQueueFiles())
	}
	return errors.Join(queue.syncStorage(), queue.closeStorageFiles())
}

func (queue *diskQueue) closeStorageFiles() error {
	return errors.Join(queue.closeReadFile(), queue.closeWriteFile())
}

func (queue *diskQueue) closeReadFile() error {
	if queue.readFile == nil {
		return nil
	}
	err := queue.readFile.Close()
	queue.readFile = nil
	return err
}

func (queue *diskQueue) closeWriteFile() error {
	if queue.writeFile == nil {
		return nil
	}
	err := queue.writeFile.Close()
	queue.writeFile = nil
	return err
}

func (queue *diskQueue) removeQueueFiles() error {
	entries, err := os.ReadDir(queue.queueDir)
	if err != nil {
		return err
	}
	var removeErr error
	for _, entry := range entries {
		if entry.IsDir() || !queue.ownsDataFile(entry.Name()) {
			continue
		}
		if err = os.Remove(filepath.Join(queue.queueDir, entry.Name())); err != nil && !errors.Is(err, os.ErrNotExist) {
			removeErr = errors.Join(removeErr, err)
		}
	}
	return errors.Join(removeErr, syncDirectory(queue.queueDir))
}

// ownsDataFile 按完整文件格式判断文件是否属于当前队列。
//
// 不能只比较 name 前缀，否则队列 a 会错误匹配队列 a.diskqueue.b。
func (queue *diskQueue) ownsDataFile(fileName string) bool {
	if fileName == queue.name+metadataFileSuffix {
		return true
	}
	temporaryPrefix := "." + queue.name + ".diskqueue.meta."
	if strings.HasPrefix(fileName, temporaryPrefix) && strings.HasSuffix(fileName, temporaryFileSuffix) {
		return true
	}
	segmentPrefix := queue.name + ".diskqueue."
	if !strings.HasPrefix(fileName, segmentPrefix) {
		return false
	}
	segmentName := strings.TrimPrefix(fileName, segmentPrefix)
	dataSuffixIndex := strings.Index(segmentName, segmentFileSuffix)
	if dataSuffixIndex <= 0 || !isUnsignedDecimal(segmentName[:dataSuffixIndex]) {
		return false
	}
	suffix := segmentName[dataSuffixIndex:]
	if suffix == segmentFileSuffix || suffix == segmentFileSuffix+badFileSuffix {
		return true
	}
	badVersionPrefix := segmentFileSuffix + badFileSuffix + "."
	return strings.HasPrefix(suffix, badVersionPrefix) && isUnsignedDecimal(strings.TrimPrefix(suffix, badVersionPrefix))
}

// isUnsignedDecimal 判断文本是否是非空的无符号十进制数字。
func isUnsignedDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (queue *diskQueue) segmentNumbers() ([]int64, error) {
	entries, err := os.ReadDir(queue.queueDir)
	if err != nil {
		return nil, err
	}
	prefix := queue.name + ".diskqueue."
	result := make([]int64, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) || !strings.HasSuffix(entry.Name(), segmentFileSuffix) {
			continue
		}
		numberText := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), prefix), segmentFileSuffix)
		fileNum, parseErr := strconv.ParseInt(numberText, 10, 64)
		if parseErr == nil && fileNum >= 0 {
			result = append(result, fileNum)
		}
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left] < result[right]
	})
	return result, nil
}

func (queue *diskQueue) nextSegmentNumber(current int64) (int64, bool) {
	segments, err := queue.segmentNumbers()
	if err != nil {
		return 0, false
	}
	for _, fileNum := range segments {
		if fileNum > current {
			return fileNum, true
		}
	}
	return 0, false
}

func (queue *diskQueue) metadataPath() string {
	return filepath.Join(queue.queueDir, queue.name+metadataFileSuffix)
}

func (queue *diskQueue) segmentPath(fileNum int64) string {
	return filepath.Join(queue.queueDir, fmt.Sprintf("%s.diskqueue.%06d.dat", queue.name, fileNum))
}

func syncDirectory(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	directory, err := os.Open(path)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
