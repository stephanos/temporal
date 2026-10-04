package architecture

import (
	"crypto/sha256"
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var memorySymbols = symbolSet(map[string][]string{
	"bytes":                     {"Compare", "Equal", "NewReader", "Split", "TrimSuffix", "Contains", "TrimSpace", "HasPrefix", "HasSuffix", "IndexByte", "Index"},
	"bytes.*Buffer":             {"Bytes", "Write", "WriteByte", "WriteString", "String", "Len", "Reset", "Grow", "Read", "ReadByte"},
	"bytes.*Reader":             {"Read", "ReadByte", "ReadAt", "Seek", "Len", "Size"},
	"encoding/hex":              {"Decode", "DecodeString", "Encode", "EncodeToString", "EncodedLen", "DecodedLen"},
	"encoding/binary.bigEndian": {"PutUint16", "PutUint32", "PutUint64", "Uint16", "Uint32", "Uint64"},
	"encoding/base64.*Encoding": {"DecodeString", "Decode", "EncodedLen", "DecodedLen", "EncodeToString", "Encode"},
	"strconv":                   {"FormatInt", "FormatUint", "ParseInt", "ParseUint", "Quote", "QuoteToASCII", "AppendQuote", "AppendInt", "AppendUint", "Atoi", "Itoa", "ParseFloat", "FormatFloat"},
	"strings":                   {"Compare", "Contains", "ContainsAny", "ContainsRune", "IndexRune", "HasPrefix", "HasSuffix", "IndexByte", "TrimPrefix", "Cut", "Fields", "Index", "Count", "EqualFold", "LastIndexByte", "TrimSuffix", "TrimSpace", "Split", "Join", "ReplaceAll", "ToLower", "ToUpper", "IndexAny", "Trim", "Clone"},
	"path":                      {"Base", "Clean", "Dir", "Join", "IsAbs", "Ext"}, "path/filepath": {"Base", "Clean", "Dir", "Join", "IsAbs", "Ext", "ToSlash", "FromSlash", "VolumeName"},
	"unicode/utf8": {"Valid", "ValidString", "RuneCount", "RuneCountInString", "DecodeRuneInString", "RuneLen"},
	"unicode":      {"IsLetter", "IsDigit", "IsSpace", "IsControl", "ToLower", "ToUpper"},
	"regexp":       {"MustCompile", "Compile"}, "regexp.*Regexp": {"MatchString", "Match", "FindStringSubmatch", "ReplaceAllString"},
	"sync.*Mutex": {"Lock", "Unlock", "TryLock"}, "sync.*RWMutex": {"Lock", "Unlock", "RLock", "RUnlock", "TryLock", "TryRLock"},
	"sync/atomic":   {"LoadUint32", "StoreUint32"},
	"sort":          {"Strings", "Ints", "Uint64s"},
	"slices":        {"BinarySearch", "Contains", "Equal", "Index", "Clone", "Sort", "IsSorted", "Delete", "Insert", "Compact", "Reverse", "Max", "Min"},
	"crypto/sha256": {"New", "Sum256", "Sum224"}, "crypto/sha256.*Digest": {"Write", "Sum", "Reset", "Size", "BlockSize"},
	"crypto/internal/fips140/sha256.*Digest": {"Write", "Sum", "Reset", "Size", "BlockSize"},
	"crypto/internal/fips140/hmac.*HMAC":     {"Write", "Sum", "Reset", "Size", "BlockSize"},
	"time.Time":                              {"Location", "Equal", "Before", "After", "Unix", "UnixNano", "IsZero", "Add", "Sub", "UTC"},
	"encoding/json.RawMessage":               {"MarshalJSON"}, "encoding/json.*RawMessage": {"UnmarshalJSON"},
	"encoding/json.Number": {"String", "Int64", "Float64"},
})

func symbolSet(groups map[string][]string) map[string]bool {
	result := map[string]bool{}
	for prefix, names := range groups {
		for _, name := range names {
			result[prefix+"."+name] = true
		}
	}
	return result
}

var callbackSummarySources = map[string]string{
	"errors/join.go":          "87da7c120f71d17e37f110ae1dc6af6493ec841a22ee825e13a6a8024a014dba",
	"errors/wrap.go":          "098116636610ae87dd80e49c4dbe6a2b6c37918d7e1af35ded005be621d68a40",
	"fmt/errors.go":           "01fd304868493a3aa1198cc79a56274930f687353e9405211ca063c1b8195ea6",
	"fmt/print.go":            "10d7cd625d83a15f70b6e61641d3cebd4e4d49d7e0f2a24d5d2b59988074d2f7",
	"encoding/json/stream.go": "065501364e4954cf1c8f1887900248d4c0983593b8c058b2fe7997d9635b793d",
	"encoding/json/encode.go": "8ff45e82c60c6e29d11fe57ce0c59d79f40ef0d53be28ac36617482a7463b217",
	"encoding/json/decode.go": "1632161a34c8286722716a48ba0b5e0c3d117a2e017e79ace676403808a16e6e",
	"time/format.go":          "f2b75d1440a46231523d3be86925b52c59c5f23657658e24514f2daa588e575d",
	"time/format_rfc3339.go":  "14b2a58fa295cca1a4f99467f5a5059418ed266a2298316124af97ffa014f271",
}

func (a *effectAnalysis) checkSummarySource(path string, pos token.Pos) bool {
	parts := strings.Split(path, "/")
	packagePath := strings.Join(parts[:len(parts)-1], "/")
	pkg := a.program.Metadata[packagePath]
	if !pkg.Standard {
		a.report("unresolved-effect", pos, "summary requires standard-library identity "+path)
		return false
	}
	data, err := os.ReadFile(filepath.Join(pkg.Dir, parts[len(parts)-1]))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != callbackSummarySources[path] {
		a.report("unresolved-effect", pos, "pinned summary source changed: "+path)
		return false
	}
	return true
}

func modeledResults(fn *types.Func) *abstractValue {
	result := tupleValue(fn.Type().(*types.Signature).Results())
	var mark func(*abstractValue)
	mark = func(v *abstractValue) {
		if v == nil {
			return
		}
		for _, typ := range v.types {
			if typ != nil && types.Identical(typ, types.Universe.Lookup("error").Type()) {
				v.builtin = true
				v.unknown = false
			}
		}
		for _, child := range v.fields {
			mark(child)
		}
	}
	mark(result)
	return result
}

func (a *effectAnalysis) standard(fn *types.Func, receiver *abstractValue, args []*abstractValue, pos token.Pos) (*abstractValue, bool) {
	result, handled := a.standardUnchecked(fn, receiver, args, pos)
	if handled && fn.Pkg() != nil && memorySummarySources[fn.Pkg().Path()] != "" {
		a.checkMemorySource(fn.Pkg().Path(), pos)
	}
	return result, handled
}

func (a *effectAnalysis) standardUnchecked(fn *types.Func, receiver *abstractValue, args []*abstractValue, pos token.Pos) (*abstractValue, bool) {
	if fn.Pkg() == nil {
		return nil, false
	}
	path := fn.Pkg().Path()
	metadata := a.program.Metadata[path]
	if !metadata.Standard {
		return nil, false
	}
	id := functionID(fn)
	result := modeledResults(fn)
	if result != nil {
		result.origin = pos
	}
	arg := func(i int) *abstractValue {
		if i < len(args) {
			return args[i]
		}
		return nil
	}
	switch {
	case path == "net" && (symbolSet(map[string][]string{"net": {"Dial", "DialTimeout", "Listen", "ListenPacket", "ListenIP", "ListenTCP", "ListenUDP", "ListenUnix", "ListenUnixgram", "DialIP", "DialTCP", "DialUDP", "DialUnix", "LookupAddr", "LookupCNAME", "LookupHost", "LookupIP", "LookupMX", "LookupNS", "LookupSRV", "LookupTXT"}, "net.*Dialer": {"Dial", "DialContext"}, "net.*ListenConfig": {"Listen", "ListenPacket"}})[id] || strings.HasPrefix(id, "net.*TCPConn.") || strings.HasPrefix(id, "net.*UDPConn.") || strings.HasPrefix(id, "net.*UnixConn.") || strings.HasPrefix(id, "net.*IPConn.") || strings.HasPrefix(id, "net.*TCPListener.") || strings.HasPrefix(id, "net.*UnixListener.")):
		a.report("host-effect", pos, id)
		return result, true
	case path == "syscall" || path == "os/exec" || path == "crypto/rand" || path == "math/rand" || path == "math/rand/v2" || strings.HasPrefix(id, "os.*File.") || strings.HasPrefix(id, "os.*Root."):
		a.report("host-effect", pos, id)
		return result, true
	case symbolSet(map[string][]string{"os": {"Open", "OpenFile", "Create", "ReadFile", "WriteFile", "Stat", "Lstat", "ReadDir", "Mkdir", "MkdirAll", "Remove", "RemoveAll", "Rename", "Getenv", "LookupEnv", "Environ", "Setenv", "Unsetenv", "Getwd", "Chdir", "Exit", "FindProcess", "StartProcess", "Executable", "UserHomeDir", "TempDir", "Pipe"}, "time": {"Now", "Parse", "Sleep", "After", "AfterFunc", "Tick", "NewTimer", "NewTicker", "LoadLocation", "LoadLocationFromTZData"}, "runtime": {"nanotime", "walltime", "fastrand", "rand", "startTheWorld", "newproc"}})[id]:
		a.report("host-effect", pos, id)
		return result, true
	case id == "time.ParseInLocation":
		if !a.checkSummarySource("time/format.go", pos) || !a.checkSummarySource("time/format_rfc3339.go", pos) {
			return result, true
		}
		if arg(2) == nil || !arg(2).utc {
			a.report("host-effect", pos, "time.ParseInLocation requires proven time.UTC")
		}
		return result, true
	case memorySymbols[id]:
		if !a.checkMemorySource(path, pos) {
			return result, true
		}
		if id == "crypto/sha256.New" {
			result.builtin = true
			result.unknown = false
		}
		switch id {
		case "slices.Clone":
			if arg(0) != nil {
				copy := *arg(0)
				copy.origin = pos
				result = &copy
			}
		case "slices.Insert":
			if arg(0) != nil {
				result = arg(0)
				for _, inserted := range args[2:] {
					result.elements = join(result.elements, inserted)
					if inserted != nil && inserted.elements != nil {
						result.elements = join(result.elements, inserted.elements)
					}
				}
			}
		case "slices.Delete", "slices.Compact":
			if arg(0) != nil {
				result = arg(0)
			}
		}
		return result, true
	case id == "errors.New":
		result.builtin = true
		result.unknown = false
		a.builtinErrorType(result, "errors", "errorString")
		return result, true
	case id == "errors.Join":
		a.checkSummarySource("errors/join.go", pos)
		result.builtin = true
		result.unknown = false
		a.builtinErrorType(result, "errors", "joinError")
		result.elements = nil
		for _, v := range args {
			if v != nil && v.builtin && v.elements != nil {
				v = v.elements
			}
			result.elements = join(result.elements, v)
		}
		return result, true
	case id == "errors.*joinError.Error":
		a.checkSummarySource("errors/join.go", pos)
		if receiver == nil || receiver.elements == nil {
			a.report("unresolved-effect", pos, "unknown joined error children")
		} else {
			a.implicit(receiver.elements, []string{"Error"}, nil, pos, map[types.Type]bool{})
		}
		return result, true
	case id == "errors.*joinError.Unwrap" || id == "fmt.*wrapError.Unwrap" || id == "fmt.*wrapErrors.Unwrap":
		if receiver == nil {
			a.report("unresolved-effect", pos, "unknown wrapped error children")
			return result, true
		}
		return receiver.elements, true
	case id == "fmt.*wrapError.Error" || id == "fmt.*wrapErrors.Error" || id == "errors.*errorString.Error":
		return result, true
	case id == "errors.Is" || id == "errors.As" || id == "errors.Unwrap":
		a.checkSummarySource("errors/wrap.go", pos)
		if id == "errors.Unwrap" {
			return a.singleUnwrap(arg(0), pos), true
		}
		a.errorCallbacks(arg(0), args[1:], pos, map[string]bool{}, fn.Name())
		return result, true
	case path == "fmt" && symbolSet(map[string][]string{"fmt": {"Errorf", "Sprintf", "Sprint", "Sprintln", "Append", "Appendf", "Appendln", "Fprint", "Fprintf", "Fprintln", "Print", "Printf", "Println"}})[id]:
		if !a.checkSummarySource("fmt/print.go", pos) {
			return result, true
		}
		start := 0
		if id == "fmt.Errorf" {
			a.checkSummarySource("fmt/errors.go", pos)
			start = 1
			result.builtin = true
			result.unknown = false
			if arg(0) == nil || arg(0).text == "" || strings.Contains(arg(0).text, "w") {
				a.builtinErrorType(result, "fmt", "wrapError")
			} else {
				a.builtinErrorType(result, "errors", "errorString")
			}
		} else if id == "fmt.Sprintf" || id == "fmt.Printf" {
			start = 1
		} else if id == "fmt.Appendf" {
			start = 2
		} else if id == "fmt.Append" || id == "fmt.Appendln" {
			start = 1
		}
		if strings.HasPrefix(id, "fmt.F") {
			a.implicit(arg(0), []string{"Write"}, nil, pos, map[types.Type]bool{})
			start = 1
			if id == "fmt.Fprintf" {
				start = 2
			}
		}
		if id == "fmt.Print" || id == "fmt.Printf" || id == "fmt.Println" {
			a.report("host-effect", pos, id)
		}
		format := arg(0)
		if id == "fmt.Fprintf" || id == "fmt.Appendf" {
			format = arg(1)
		}
		formatted := id == "fmt.Errorf" || id == "fmt.Sprintf" || id == "fmt.Appendf" || id == "fmt.Fprintf" || id == "fmt.Printf"
		operands := formatOperands("", len(args)-start, false)
		if !formatted {
			for i := range operands {
				operands[i].names = []string{"Format", "Error", "String"}
				operands[i].wrap = false
			}
		} else if format != nil && format.text != "" {
			operands = formatOperands(format.text, len(args)-start, true)
		}
		for _, operand := range operands {
			if operand.index < 0 || operand.index >= len(args)-start {
				continue
			}
			v := args[start+operand.index]
			if len(operand.names) > 0 {
				a.implicit(v, operand.names, nil, pos, map[types.Type]bool{})
			}
			if id == "fmt.Errorf" && operand.wrap {
				result.elements = join(result.elements, v)
			}
		}
		return result, true
	case id == "encoding/json.NewEncoder" || id == "encoding/json.NewDecoder":
		a.checkSummarySource("encoding/json/stream.go", pos)
		result.fields = map[string]*abstractValue{"transport": arg(0)}
		return result, true
	case id == "encoding/json.*Encoder.Encode":
		a.checkSummarySource("encoding/json/encode.go", pos)
		a.checkSummarySource("encoding/json/stream.go", pos)
		if receiver != nil {
			result = join(result, callbackError(a.implicit(receiver.fields["transport"], []string{"Write"}, nil, pos, map[types.Type]bool{})))
		} else {
			a.report("unresolved-effect", pos, "unknown JSON encoder writer")
		}
		result = join(result, callbackError(a.implicit(arg(0), []string{"MarshalJSON", "MarshalText"}, nil, pos, map[types.Type]bool{})))
		return result, true
	case id == "encoding/json.Marshal" || id == "encoding/json.MarshalIndent":
		a.checkSummarySource("encoding/json/encode.go", pos)
		result.fields["1"] = join(result.fields["1"], callbackError(a.implicit(arg(0), []string{"MarshalJSON", "MarshalText"}, nil, pos, map[types.Type]bool{})))
		return result, true
	case id == "encoding/json.Unmarshal":
		a.checkSummarySource("encoding/json/decode.go", pos)
		result = join(result, callbackError(a.implicit(arg(1), []string{"UnmarshalJSON", "UnmarshalText"}, nil, pos, map[types.Type]bool{})))
		a.decoded(arg(1))
		return result, true
	case symbolSet(map[string][]string{"encoding/json.*Decoder": {"Decode", "Token", "More"}})[id]:
		a.checkSummarySource("encoding/json/stream.go", pos)
		a.checkSummarySource("encoding/json/decode.go", pos)
		if receiver != nil {
			readError := callbackError(a.implicit(receiver.fields["transport"], []string{"Read"}, nil, pos, map[types.Type]bool{}))
			if id == "encoding/json.*Decoder.Decode" {
				result = join(result, readError)
			} else if id == "encoding/json.*Decoder.Token" {
				result.fields["1"] = join(result.fields["1"], readError)
			}
		} else {
			a.report("unresolved-effect", pos, "unknown JSON decoder reader")
		}
		if id == "encoding/json.*Decoder.Decode" {
			result = join(result, callbackError(a.implicit(arg(0), []string{"UnmarshalJSON", "UnmarshalText"}, nil, pos, map[types.Type]bool{})))
			a.decoded(arg(0))
		} else if result != nil {
			if result.fields["0"] != nil {
				result.fields["0"].builtin = true
				result.fields["0"].unknown = false
			}
		}
		return result, true
	case symbolSet(map[string][]string{"encoding/json.*Decoder": {"UseNumber", "DisallowUnknownFields"}, "encoding/json.*Encoder": {"SetEscapeHTML"}})[id]:
		return result, true
	case path == "sort" && symbolSet(map[string][]string{"sort": {"Slice", "SliceStable", "SliceIsSorted"}})[id]:
		a.invoke(arg(1), []*abstractValue{valueOf(types.Typ[types.Int]), valueOf(types.Typ[types.Int])}, pos)
		return result, true
	case path == "slices" && symbolSet(map[string][]string{"slices": {"SortFunc", "SortStableFunc", "EqualFunc", "IsSortedFunc", "BinarySearchFunc", "ContainsFunc", "IndexFunc", "CompactFunc"}})[id]:
		for _, v := range args {
			if len(v.functions) > 0 || v.unknown {
				a.invoke(v, []*abstractValue{elementValue(arg(0)), elementValue(arg(0))}, pos)
			}
		}
		return result, true
	case path == "container/heap" || id == "sort.Sort" || id == "sort.Stable":
		a.implicit(arg(0), []string{"Len", "Less", "Swap", "Push", "Pop"}, args[1:], pos, map[types.Type]bool{})
		return result, true
	case id == "crypto/hmac.New":
		hash := a.invoke(arg(0), nil, pos)
		result.builtin = true
		result.unknown = false
		result.elements = hash
		return result, true
	case path == "reflect" && symbolSet(map[string][]string{"reflect": {"ValueOf", "TypeOf"}, "reflect.Value": {"IsValid", "Kind", "IsNil", "Elem", "Type", "Pointer", "MapRange", "Len", "Index", "NumField", "Field", "String", "CanInterface", "Interface"}, "reflect.*MapIter": {"Next", "Key", "Value"}})[id]:
		if id == "reflect.ValueOf" {
			result.fields = map[string]*abstractValue{"represented": arg(0)}
		} else if receiver != nil {
			result.fields = receiver.fields
		}
		return result, true
	}
	if receiver != nil && receiver.builtin && symbolSet(map[string][]string{"io.Writer": {"Write"}, "hash.Hash": {"Sum", "Reset", "Size", "BlockSize"}, "error": {"Error"}})[id] {
		return result, true
	}
	if receiver != nil && receiver.builtin && fn.Name() == "Error" {
		if receiver.elements != nil {
			a.implicit(receiver.elements, []string{"Error"}, nil, pos, map[types.Type]bool{})
		}
		return result, true
	}
	return nil, false
}

func (a *effectAnalysis) checkMemorySource(path string, pos token.Pos) bool {
	if a.checkedMemory == nil {
		a.checkedMemory = map[string]bool{}
	}
	if a.checkedMemory[path] {
		return true
	}
	metadata := a.program.Metadata[path]
	if !metadata.Standard {
		a.report("unresolved-effect", pos, "memory summary requires standard-library identity "+path)
		return false
	}
	entries, err := os.ReadDir(metadata.Dir)
	if err != nil {
		a.report("unresolved-effect", pos, "cannot inspect memory summary source "+path)
		return false
	}
	digest := sha256.New()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(metadata.Dir, entry.Name()))
		if err != nil {
			a.report("unresolved-effect", pos, "cannot read memory summary source "+path)
			return false
		}
		fmt.Fprintf(digest, "%x  %s\n", sha256.Sum256(data), entry.Name())
	}
	if fmt.Sprintf("%x", digest.Sum(nil)) != memorySummarySources[path] {
		a.report("unresolved-effect", pos, "pinned memory summary source changed: "+path)
		return false
	}
	a.checkedMemory[path] = true
	return true
}

func (a *effectAnalysis) builtinErrorType(v *abstractValue, path, name string) {
	pkg := a.program.Packages[path]
	if pkg == nil {
		return
	}
	if obj := pkg.Types.Scope().Lookup(name); obj != nil {
		v.types = []types.Type{types.NewPointer(obj.Type())}
	}
}

func elementValue(v *abstractValue) *abstractValue {
	if v == nil {
		return nil
	}
	if v.elements != nil {
		return v.elements
	}
	for _, typ := range v.types {
		switch typ := typ.Underlying().(type) {
		case *types.Slice:
			return valueOf(typ.Elem())
		case *types.Array:
			return valueOf(typ.Elem())
		}
	}
	return nil
}

func callbackError(v *abstractValue) *abstractValue {
	if v == nil {
		return nil
	}
	if child := v.fields["1"]; child != nil {
		return child
	}
	return v
}

func (a *effectAnalysis) invoke(v *abstractValue, args []*abstractValue, pos token.Pos) *abstractValue {
	var result *abstractValue
	if v != nil {
		for _, fn := range v.functions {
			result = join(result, a.call(fn, args, pos))
		}
		if len(v.functions) > 0 && !v.unknown {
			return result
		}
	}
	a.report("unresolved-effect", pos, "unknown invoked callback binding")
	return result
}

func (a *effectAnalysis) decoded(v *abstractValue) {
	if v == nil {
		return
	}
	for _, typ := range v.types {
		if pointer, ok := typ.(*types.Pointer); ok {
			if _, ok := pointer.Elem().Underlying().(*types.Interface); ok {
				v.builtin = true
				v.unknown = false
				if pointee := v.fields["$pointee"]; pointee != nil {
					pointee.builtin = true
					pointee.unknown = false
				}
			}
		}
	}
}

func (a *effectAnalysis) implicit(v *abstractValue, names []string, args []*abstractValue, pos token.Pos, seen map[types.Type]bool) *abstractValue {
	return a.implicitAddressable(v, names, args, pos, seen, false)
}

func (a *effectAnalysis) implicitAddressable(v *abstractValue, names []string, args []*abstractValue, pos token.Pos, seen map[types.Type]bool, addressable bool) *abstractValue {
	if v == nil {
		a.report("unresolved-effect", pos, "unknown implicit callback receiver")
		return nil
	}
	if a.implicitActive == nil {
		a.implicitActive = map[*abstractValue]map[string]bool{}
	}
	context := fmt.Sprint(addressable) + strings.Join(names, ",") + bindingKey(args)
	if a.implicitActive[v] == nil {
		a.implicitActive[v] = map[string]bool{}
	}
	if a.implicitActive[v][context] {
		return nil
	}
	a.implicitActive[v][context] = true
	defer delete(a.implicitActive[v], context)
	if v.builtin {
		for _, typ := range v.types {
			if types.TypeString(typ, func(p *types.Package) string { return p.Path() }) == "*errors.joinError" && v.elements != nil {
				return a.implicit(v.elements, names, args, pos, seen)
			}
		}
		return nil
	}
	var result *abstractValue
	for _, typ := range v.types {
		if typ == nil || seen[typ] {
			continue
		}
		seen[typ] = true
		if _, ok := typ.Underlying().(*types.Interface); ok {
			if v.unknown {
				a.report("unresolved-effect", pos, "unknown dynamic callback receiver "+types.TypeString(typ, nil))
			}
			continue
		}
		methods := types.NewMethodSet(typ)
		receiver := v
		if addressable && (slices.Contains(names, "MarshalJSON") || slices.Contains(names, "MarshalText") || slices.Contains(names, "UnmarshalJSON") || slices.Contains(names, "UnmarshalText")) {
			if _, pointer := typ.(*types.Pointer); !pointer {
				methods = types.NewMethodSet(types.NewPointer(typ))
				copy := *v
				copy.types = []types.Type{types.NewPointer(typ)}
				copy.fields = map[string]*abstractValue{"$pointee": v}
				for name, child := range v.fields {
					copy.fields[name] = child
				}
				receiver = &copy
			}
		}
		matched := false
		for _, name := range names {
			if matched {
				break
			}
			for i := 0; i < methods.Len(); i++ {
				fn := methods.At(i).Obj().(*types.Func)
				if fn.Name() != name {
					continue
				}
				matched = true
				callArgs := args
				if name == "Format" {
					state := valueOf(types.NewInterfaceType(nil, nil))
					state.builtin = true
					state.unknown = false
					callArgs = []*abstractValue{state, valueOf(types.Typ[types.Rune])}
				}
				if returned, handled := a.standard(fn, receiver, callArgs, pos); handled {
					result = join(result, returned)
				} else {
					if decl := a.program.Functions[functionID(fn)]; decl != nil {
						result = join(result, a.call(&boundFunction{fn: decl, pkg: decl.pkg, receiver: receiver}, callArgs, pos))
					} else {
						a.report("unresolved-effect", pos, "implicit call "+functionID(fn))
					}
				}
			}
		}
		if matched {
			continue
		}
		if len(names) == 1 && names[0] != "MarshalJSON" && names[0] != "MarshalText" && names[0] != "UnmarshalJSON" && names[0] != "UnmarshalText" {
			continue
		}
		switch underlying := typ.Underlying().(type) {
		case *types.Pointer:
			child := *v
			child.types = []types.Type{underlying.Elem()}
			result = join(result, a.implicitAddressable(&child, names, args, pos, seen, true))
		case *types.Slice:
			child := elementValue(v)
			if child != nil {
				result = join(result, a.implicitAddressable(child, names, args, pos, seen, true))
			}
		case *types.Array:
			child := elementValue(v)
			if child != nil {
				result = join(result, a.implicitAddressable(child, names, args, pos, seen, addressable))
			}
		case *types.Map:
			result = join(result, a.implicit(valueOf(underlying.Key()), names, args, pos, seen))
			if v.elements != nil {
				result = join(result, a.implicit(v.elements, names, args, pos, map[types.Type]bool{}))
			} else {
				result = join(result, a.implicit(valueOf(underlying.Elem()), names, args, pos, seen))
			}
		case *types.Struct:
			for i := 0; i < underlying.NumFields(); i++ {
				field := underlying.Field(i)
				if !field.Exported() {
					continue
				}
				child := v.fields[field.Name()]
				childSeen := map[types.Type]bool{}
				if child == nil {
					child = valueOf(field.Type())
					childSeen = seen
				}
				result = join(result, a.implicitAddressable(child, names, args, pos, childSeen, addressable))
			}
		}
	}
	return result
}

func (a *effectAnalysis) errorCallbacks(v *abstractValue, args []*abstractValue, pos token.Pos, seen map[string]bool, names ...string) {
	if v == nil {
		a.report("unresolved-effect", pos, "unknown error callback receiver")
		return
	}
	key := bindingKey([]*abstractValue{v})
	if seen[key] {
		return
	}
	seen[key] = true
	if v.builtin {
		if v.elements != nil {
			a.errorCallbacks(v.elements, args, pos, seen, names...)
		}
		return
	}
	a.implicit(v, names, args, pos, map[types.Type]bool{})
	child := a.implicit(v, []string{"Unwrap"}, nil, pos, map[types.Type]bool{})
	if child != nil {
		if child.elements != nil {
			a.errorCallbacks(child.elements, args, pos, seen, names...)
		} else {
			a.errorCallbacks(child, args, pos, seen, names...)
		}
	}
}

func (a *effectAnalysis) singleUnwrap(v *abstractValue, pos token.Pos) *abstractValue {
	if v == nil {
		return nil
	}
	var result *abstractValue
	for _, typ := range v.types {
		if v.builtin {
			if types.TypeString(typ, func(pkg *types.Package) string { return pkg.Path() }) == "*fmt.wrapError" {
				result = join(result, v.elements)
			}
			continue
		}
		if _, ok := typ.Underlying().(*types.Interface); ok && v.unknown {
			a.report("unresolved-effect", pos, "unknown dynamic Unwrap receiver")
		}
		methods := types.NewMethodSet(typ)
		for i := 0; i < methods.Len(); i++ {
			method := methods.At(i).Obj().(*types.Func)
			signature := method.Type().(*types.Signature)
			if method.Name() != "Unwrap" || signature.Params().Len() != 0 || signature.Results().Len() != 1 ||
				!types.Identical(signature.Results().At(0).Type(), types.Universe.Lookup("error").Type()) {
				continue
			}
			child := *v
			child.types = []types.Type{typ}
			result = join(result, a.implicit(&child, []string{"Unwrap"}, nil, pos, map[types.Type]bool{}))
		}
	}
	return result
}

type formatOperand struct {
	index int
	names []string
	wrap  bool
}

func formatOperands(format string, count int, known bool) []formatOperand {
	if !known {
		var result []formatOperand
		for i := 0; i < count; i++ {
			result = append(result, formatOperand{i, []string{"Format", "GoString", "Error", "String"}, true})
		}
		return result
	}
	var result []formatOperand
	argument := 0
	reordered := false
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i >= len(format) {
			break
		}
		if format[i] == '%' {
			continue
		}
		sharp := false
		for i < len(format) {
			c := format[i]
			if c == '#' {
				sharp = true
			}
			if strings.ContainsRune("#+- 0", rune(c)) {
				i++
				continue
			}
			if c == '[' {
				end := strings.IndexByte(format[i:], ']')
				if end < 0 {
					break
				}
				index, err := strconv.Atoi(format[i+1 : i+end])
				if err != nil {
					break
				}
				argument = index - 1
				reordered = true
				i += end + 1
				continue
			}
			if c == '*' {
				argument++
				i++
				continue
			}
			if c == '.' || c >= '0' && c <= '9' {
				i++
				continue
			}
			break
		}
		if i >= len(format) {
			break
		}
		verb := format[i]
		names := []string{"Format"}
		if verb == 'T' || verb == 'p' {
			names = nil
		} else if verb == 'v' && sharp {
			names = append(names, "GoString")
		} else if strings.ContainsRune("vsxXqw", rune(verb)) {
			names = append(names, "Error", "String")
		}
		result = append(result, formatOperand{argument, names, verb == 'w'})
		argument++
	}
	if !reordered {
		for argument < count {
			result = append(result, formatOperand{argument, []string{"Format", "Error", "String"}, false})
			argument++
		}
	}
	return result
}

func typeOnlyFormat(format string) bool {
	if format == "" {
		return false
	}
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			continue
		}
		i++
		if i >= len(format) {
			return false
		}
		if format[i] == '%' {
			continue
		}
		for i < len(format) && strings.ContainsRune("#+- .0123456789[]", rune(format[i])) {
			i++
		}
		if i >= len(format) || (format[i] != 'T' && format[i] != 'p') {
			return false
		}
	}
	return true
}
