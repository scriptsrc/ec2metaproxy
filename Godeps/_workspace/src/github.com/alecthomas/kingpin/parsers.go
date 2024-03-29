package kingpin

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/flyinprogrammer/ec2metaproxy/Godeps/_workspace/src/github.com/alecthomas/units"
)

type Settings interface {
	SetValue(value Value)
}

type parserMixin struct {
	value    Value
	required bool
}

func (p *parserMixin) SetValue(value Value) {
	p.value = value
}

// String sets the parser to a string parser.
func (p *parserMixin) String() (target *string) {
	target = new(string)
	p.StringVar(target)
	return
}

// Strings appends multiple occurrences to a string slice.
func (p *parserMixin) Strings() (target *[]string) {
	target = new([]string)
	p.StringsVar(target)
	return
}

// StringMap provides key=value parsing into a map.
func (p *parserMixin) StringMap() (target *map[string]string) {
	target = &(map[string]string{})
	p.StringMapVar(target)
	return
}

// Bool sets the parser to a boolean parser. Supports --no-<X> to disable the flag.
func (p *parserMixin) Bool() (target *bool) {
	target = new(bool)
	p.BoolVar(target)
	return
}

// Int sets the parser to an int parser.
func (p *parserMixin) Int() (target *int) {
	target = new(int)
	p.IntVar(target)
	return
}

// Int64 parses an int64
func (p *parserMixin) Int64() (target *int64) {
	target = new(int64)
	p.Int64Var(target)
	return
}

// Uint64 parses a uint64
func (p *parserMixin) Uint64() (target *uint64) {
	target = new(uint64)
	p.Uint64Var(target)
	return
}

// Float sets the parser to a float64 parser.
func (p *parserMixin) Float() (target *float64) {
	target = new(float64)
	p.FloatVar(target)
	return
}

// Duration sets the parser to a time.Duration parser.
func (p *parserMixin) Duration() (target *time.Duration) {
	target = new(time.Duration)
	p.DurationVar(target)
	return
}

// Bytes parses numeric byte units. eg. 1.5KB
func (p *parserMixin) Bytes() (target *units.Base2Bytes) {
	target = new(units.Base2Bytes)
	p.BytesVar(target)
	return
}

// IP sets the parser to a net.IP parser.
func (p *parserMixin) IP() (target *net.IP) {
	target = new(net.IP)
	p.IPVar(target)
	return
}

// TCP (host:port) address.
func (p *parserMixin) TCP() (target **net.TCPAddr) {
	target = new(*net.TCPAddr)
	p.TCPVar(target)
	return
}

// TCPVar (host:port) address.
func (p *parserMixin) TCPVar(target **net.TCPAddr) {
	p.SetValue(newTCPAddrValue(target))
}

// TCP (host:port) address list.
func (p *parserMixin) TCPList() (target *[]*net.TCPAddr) {
	target = new([]*net.TCPAddr)
	p.TCPListVar(target)
	return
}

// TCPVar (host:port) address list.
func (p *parserMixin) TCPListVar(target *[]*net.TCPAddr) {
	p.SetValue(newTCPAddrsValue(target))
}

// ExistingFile sets the parser to one that requires and returns an existing file.
func (p *parserMixin) ExistingFile() (target *string) {
	target = new(string)
	p.ExistingFileVar(target)
	return
}

// ExistingDir sets the parser to one that requires and returns an existing directory.
func (p *parserMixin) ExistingDir() (target *string) {
	target = new(string)
	p.ExistingDirVar(target)
	return
}

// File returns an os.File against an existing file.
func (p *parserMixin) File() (target **os.File) {
	target = new(*os.File)
	p.FileVar(target)
	return
}

// File attempts to open a File with os.OpenFile(flag, perm).
func (p *parserMixin) OpenFile(flag int, perm os.FileMode) (target **os.File) {
	target = new(*os.File)
	p.OpenFileVar(target, flag, perm)
	return
}

// URL provides a valid, parsed url.URL.
func (p *parserMixin) URL() (target **url.URL) {
	target = new(*url.URL)
	p.URLVar(target)
	return
}

// String sets the parser to a string parser.
func (p *parserMixin) StringVar(target *string) {
	p.SetValue(newStringValue("", target))
}

// Strings appends multiple occurrences to a string slice.
func (p *parserMixin) StringsVar(target *[]string) {
	p.SetValue(newStringsValue(target))
}

// StringMap provides key=value parsing into a map.
func (p *parserMixin) StringMapVar(target *map[string]string) {
	p.SetValue(newStringMapValue(target))
}

// Bool sets the parser to a boolean parser. Supports --no-<X> to disable the flag.
func (p *parserMixin) BoolVar(target *bool) {
	p.SetValue(newBoolValue(false, target))
}

// Int sets the parser to an int parser.
func (p *parserMixin) IntVar(target *int) {
	p.SetValue(newIntValue(0, target))
}

// Int64 parses an int64
func (p *parserMixin) Int64Var(target *int64) {
	p.SetValue(newInt64Value(0, target))
}

// Uint64 parses a uint64
func (p *parserMixin) Uint64Var(target *uint64) {
	p.SetValue(newUint64Value(0, target))
}

// Float sets the parser to a float64 parser.
func (p *parserMixin) FloatVar(target *float64) {
	p.SetValue(newFloat64Value(0, target))
}

// Duration sets the parser to a time.Duration parser.
func (p *parserMixin) DurationVar(target *time.Duration) {
	p.SetValue(newDurationValue(time.Duration(0), target))
}

// BytesVar parses numeric byte units. eg. 1.5KB
func (p *parserMixin) BytesVar(target *units.Base2Bytes) {
	p.SetValue(newBytesValue(units.Base2Bytes(0), target))
}

// IP sets the parser to a net.IP parser.
func (p *parserMixin) IPVar(target *net.IP) {
	p.SetValue(newIPValue(target))
}

// ExistingFile sets the parser to one that requires and returns an existing file.
func (p *parserMixin) ExistingFileVar(target *string) {
	p.SetValue(newFileStatValue(target, func(s os.FileInfo) error {
		if s.IsDir() {
			return fmt.Errorf("'%s' is a directory", s.Name())
		}
		return nil
	}))
}

// ExistingDir sets the parser to one that requires and returns an existing directory.
func (p *parserMixin) ExistingDirVar(target *string) {
	p.SetValue(newFileStatValue(target, func(s os.FileInfo) error {
		if !s.IsDir() {
			return fmt.Errorf("'%s' is a file", s.Name())
		}
		return nil
	}))
}

// FileVar opens an existing file.
func (p *parserMixin) FileVar(target **os.File) {
	p.SetValue(newFileValue(target, os.O_RDONLY, 0))
}

// OpenFileVar calls os.OpenFile(flag, perm)
func (p *parserMixin) OpenFileVar(target **os.File, flag int, perm os.FileMode) {
	p.SetValue(newFileValue(target, flag, perm))
}

// URL provides a valid, parsed url.URL.
func (p *parserMixin) URLVar(target **url.URL) {
	p.SetValue(newURLValue(target))
}

// URLList provides a parsed list of url.URL values.
func (p *parserMixin) URLList() (target *[]*url.URL) {
	target = new([]*url.URL)
	p.URLListVar(target)
	return
}

// URLListVar provides a parsed list of url.URL values.
func (p *parserMixin) URLListVar(target *[]*url.URL) {
	p.SetValue(newURLListValue(target))
}

// Enum allows a value from a set of options.
func (p *parserMixin) Enum(options ...string) (target *string) {
	target = new(string)
	p.EnumVar(&target, options...)
	return
}

// EnumVar allows a value from a set of options.
func (p *parserMixin) EnumVar(target **string, options ...string) {
	p.SetValue(newEnumFlag(target, options...))
}
