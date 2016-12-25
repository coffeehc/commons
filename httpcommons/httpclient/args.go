package httpclient

import (
	"github.com/valyala/fasthttp"
	"io"
)

type _Args struct {
	args *fasthttp.Args
}

func (args *_Args) getFastHTTPArgs() *fasthttp.Args {
	return args.args
}

func (args *_Args) Reset() {
	args.args.Reset()
}
func (args *_Args) CopyTo(dst Args) {
	args.args.CopyTo(dst.getFastHTTPArgs())
}
func (args *_Args) VisitAll(f func(key, value []byte)) {
	args.args.VisitAll(f)
}
func (args *_Args) Len() int {
	return args.args.Len()
}
func (args *_Args) Parse(s string) {
	args.args.Parse(s)
}
func (args *_Args) ParseBytes(b []byte) {
	args.args.ParseBytes(b)
}
func (args *_Args) String() string {
	return args.args.String()
}
func (args *_Args) QueryString() []byte {
	return args.args.QueryString()
}
func (args *_Args) AppendBytes(dst []byte) []byte {
	return args.args.AppendBytes(dst)
}
func (args *_Args) WriteTo(w io.Writer) (int64, error) {
	return args.args.WriteTo(w)
}
func (args *_Args) Del(key string) {
	args.args.Del(key)
}
func (args *_Args) DelBytes(key []byte) {
	args.args.DelBytes(key)
}
func (args *_Args) Add(key, value string) {
	args.args.Add(key, value)
}
func (args *_Args) AddBytesK(key []byte, value string) {
	args.args.AddBytesK(key, value)
}
func (args *_Args) AddBytesV(key string, value []byte) {
	args.args.AddBytesV(key, value)
}
func (args *_Args) AddBytesKV(key, value []byte) {
	args.args.AddBytesKV(key, value)
}
func (args *_Args) Set(key, value string) {
	args.args.Set(key, value)
}
func (args *_Args) SetBytesK(key []byte, value string) {
	args.args.SetBytesK(key, value)
}
func (args *_Args) SetBytesV(key string, value []byte) {
	args.args.SetBytesV(key, value)
}
func (args *_Args) SetBytesKV(key, value []byte) {
	args.args.SetBytesKV(key, value)
}
func (args *_Args) Peek(key string) []byte {
	return args.args.Peek(key)
}
func (args *_Args) PeekBytes(key []byte) []byte {
	return args.args.PeekBytes(key)
}
func (args *_Args) PeekMulti(key string) [][]byte {
	return args.args.PeekMulti(key)
}
func (args *_Args) PeekMultiBytes(key []byte) [][]byte {
	return args.args.PeekMultiBytes(key)
}
func (args *_Args) Has(key string) bool {
	return args.args.Has(key)
}
func (args *_Args) HasBytes(key []byte) bool {
	return args.args.HasBytes(key)
}
func (args *_Args) GetUint(key string) (int, error) {
	return args.args.GetUint(key)
}
func (args *_Args) SetUint(key string, value int) {
	args.args.SetUint(key, value)
}
func (args *_Args) SetUintBytes(key []byte, value int) {
	args.args.SetUintBytes(key, value)
}
func (args *_Args) GetUintOrZero(key string) int {
	return args.args.GetUintOrZero(key)
}
func (args *_Args) GetUfloat(key string) (float64, error) {
	return args.args.GetUfloat(key)
}
func (args *_Args) GetUfloatOrZero(key string) float64 {
	return args.args.GetUfloatOrZero(key)
}
