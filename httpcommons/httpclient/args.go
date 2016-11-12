package httpclient

import (
	"github.com/valyala/fasthttp"
	"io"
)

type _HttpArgs struct {
	args *fasthttp.Args
}

func (this *_HttpArgs)getFastHttpAtgs() *fasthttp.Args {
	return this.args
}

func (this *_HttpArgs)Reset() {
	this.args.Reset()
}
func (this *_HttpArgs)CopyTo(dst HttpArgs) {
	this.args.CopyTo(dst.getFastHttpAtgs())
}
func (this *_HttpArgs)VisitAll(f func(key, value []byte)) {
	this.args.VisitAll(f)
}
func (this *_HttpArgs)Len() int {
	return this.args.Len()
}
func (this *_HttpArgs)Parse(s string) {
	this.args.Parse(s)
}
func (this *_HttpArgs)ParseBytes(b []byte) {
	this.args.ParseBytes(b)
}
func (this *_HttpArgs)String() string {
	return this.args.String()
}
func (this *_HttpArgs)QueryString() []byte {
	return this.args.QueryString()
}
func (this *_HttpArgs)AppendBytes(dst []byte) []byte {
	return this.args.AppendBytes(dst)
}
func (this *_HttpArgs)WriteTo(w io.Writer) (int64, error) {
	return this.args.WriteTo(w)
}
func (this *_HttpArgs)Del(key string) {
	this.args.Del(key)
}
func (this *_HttpArgs)DelBytes(key []byte) {
	this.args.DelBytes(key)
}
func (this *_HttpArgs)Add(key, value string) {
	this.args.Add(key, value)
}
func (this *_HttpArgs)AddBytesK(key []byte, value string) {
	this.args.AddBytesK(key, value)
}
func (this *_HttpArgs)AddBytesV(key string, value []byte) {
	this.args.AddBytesV(key, value)
}
func (this *_HttpArgs)AddBytesKV(key, value []byte) {
	this.args.AddBytesKV(key, value)
}
func (this *_HttpArgs)Set(key, value string) {
	this.args.Set(key, value)
}
func (this *_HttpArgs)SetBytesK(key []byte, value string) {
	this.args.SetBytesK(key, value)
}
func (this *_HttpArgs)SetBytesV(key string, value []byte) {
	this.args.SetBytesV(key, value)
}
func (this *_HttpArgs)SetBytesKV(key, value []byte) {
	this.args.SetBytesKV(key, value)
}
func (this *_HttpArgs)Peek(key string) []byte {
	return this.args.Peek(key)
}
func (this *_HttpArgs)PeekBytes(key []byte) []byte {
	return this.args.PeekBytes(key)
}
func (this *_HttpArgs)PeekMulti(key string) [][]byte {
	return this.args.PeekMulti(key)
}
func (this *_HttpArgs)PeekMultiBytes(key []byte) [][]byte {
	return this.args.PeekMultiBytes(key)
}
func (this *_HttpArgs)Has(key string) bool {
	return this.args.Has(key)
}
func (this *_HttpArgs)HasBytes(key []byte) bool {
	return this.args.HasBytes(key)
}
func (this *_HttpArgs)GetUint(key string) (int, error) {
	return this.args.GetUint(key)
}
func (this *_HttpArgs)SetUint(key string, value int) {
	this.args.SetUint(key, value)
}
func (this *_HttpArgs)SetUintBytes(key []byte, value int) {
	this.args.SetUintBytes(key, value)
}
func (this *_HttpArgs)GetUintOrZero(key string) int {
	return this.args.GetUintOrZero(key)
}
func (this *_HttpArgs)GetUfloat(key string) (float64, error) {
	return this.args.GetUfloat(key)
}
func (this *_HttpArgs)GetUfloatOrZero(key string) float64 {
	return this.args.GetUfloatOrZero(key)
}
