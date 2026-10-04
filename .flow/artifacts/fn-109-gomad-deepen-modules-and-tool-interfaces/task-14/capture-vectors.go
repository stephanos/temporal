package main
import ("encoding/hex"; "encoding/json"; "fmt"; "internal/gomadmodelwire")
type vector struct { Domain string; Operation int; Request string; Response string }
func main() {
var vectors []vector
for op:=1;op<=13;op++ {
r:=gomadmodelwire.Request{Model:gomadmodelwire.ModelNetwork,Operation:gomadmodelwire.Operation(op)}
s:=gomadmodelwire.Response{}
switch op {
case 1,2:r.String1="tcp4";r.String2="127.0.0.1";r.Int1=32123;if op==2 {r.Int2=123456789}
s.Handle=17;s.String1="127.0.0.1";s.Int1=32123;if op==2 {s.String2="127.0.0.2";s.Int2=32124}
case 3:r.Handle=7;s.Handle=17;s.String1="127.0.0.1";s.Int1=32123;s.String2="127.0.0.2";s.Int2=32124
case 6:r.Handle=7;r.Uint1=9;s.Uint1=2;s.Data=[]byte("ab");s.Error=gomadmodelwire.WireError{Code:gomadmodelwire.ErrorEOF,Message:"EOF"}
case 7:r.Handle=7;r.Data=[]byte("abcd");s.Uint1=2;s.Error=gomadmodelwire.WireError{Code:gomadmodelwire.ErrorEOF,Message:"EOF"}
default:r.Handle=7;if op==5 || op>=11 {r.Int1=123456789};s.Error=gomadmodelwire.WireError{Code:gomadmodelwire.ErrorESTALE,Message:"stale NFS file handle"}
}
vectors=append(vectors,capture("network",op,r,s))
}
for op:=1;op<=28;op++ {
r:=gomadmodelwire.Request{Model:gomadmodelwire.ModelVolume,Operation:gomadmodelwire.Operation(op)}
s:=gomadmodelwire.Response{}
entry:=gomadmodelwire.Entry{Name:"state",Mode:416,Kind:1,ModTime:123456789,Data:[]byte("ab")}
if op<=11 { r.String1="/state" }
if op>=13 {r.Handle=7}
switch op {
case 1:s.String1="/state";s.String2="state"
case 2,3,9:r.Uint1=416
case 4,22:s.Entries=[]gomadmodelwire.Entry{entry}
case 5:r.Uint1=416;r.Flags=63;s.Handle=17;s.String1="/state"
case 6:r.String2="/next"
case 10,19:r.Int1=123456789
case 12:s.String1="/root"
case 13,14:r.Uint1=9;if op==14 {r.Int1=19};s.Uint1=2;s.Data=[]byte("ab");s.Error=gomadmodelwire.WireError{Code:gomadmodelwire.ErrorEOF,Message:"EOF"}
case 15,16:r.Data=[]byte("abcd");if op==16 {r.Int1=19};s.Uint1=2;s.Error=gomadmodelwire.WireError{Code:gomadmodelwire.ErrorEOF,Message:"EOF"}
case 17:r.Int1=19
case 18:r.Uint1=416
case 21:r.Int1=19;r.Int2=2;s.Int1=23
case 23:r.Int1=3;s.Entries=[]gomadmodelwire.Entry{entry};s.Error=gomadmodelwire.WireError{Code:gomadmodelwire.ErrorEOF,Message:"EOF"}
case 26:r.Int1=19;r.Uint1=9;s.Handle=17
case 27:s.Data=[]byte("ab")
}
vectors=append(vectors,capture("volume",op,r,s))
}
b,e:=json.MarshalIndent(vectors,"","  ");if e!=nil {panic(e)};fmt.Println(string(b))
}
func capture(d string,op int,r gomadmodelwire.Request,s gomadmodelwire.Response) vector {
a,e:=gomadmodelwire.EncodeRequest(r);if e!=nil {panic(e)}
b,e:=gomadmodelwire.EncodeResponse(s);if e!=nil {panic(e)}
return vector{d,op,hex.EncodeToString(a),hex.EncodeToString(b)}
}


