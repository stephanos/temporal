package gomadio

import "internal/gomadsim"

type ProcessNetworkOperationForTest = processNetworkOperation
type ProcessNetworkCommandForTest = processNetworkCommand
type ProcessNetworkResultForTest = processNetworkResult

var EncodeProcessNetworkCommandForTest = encodeProcessNetworkCommand
var DecodeProcessNetworkCommandForTest = decodeProcessNetworkCommand
var EncodeProcessNetworkResultForTest = encodeProcessNetworkResponse
var DecodeProcessNetworkResultForTest = decodeProcessNetworkResult

var EncodeProcessNetworkErrorForTest = encodeProcessNetworkError
var DecodeProcessNetworkErrorForTest = decodeProcessNetworkError

func ApplyProcessNetworkOperationForTest(domain uint64, command processNetworkCommand) processNetworkResult {
	return applyProcessNetworkOperation(gomadsim.NetworkDomain{Token: domain}, command)
}

func RegisterProcessNetworkResourceForTest(domain uint64, listener bool) uint64 {
	resource := processNetworkResource{domain: domain}
	if listener {
		resource.listener = &Listener{}
	} else {
		resource.conn = &Conn{}
	}
	handle, err := registerProcessNetworkResource(resource)
	if err != nil {
		panic(err)
	}
	return handle
}

var RemoveProcessNetworkResourceForTest = removeProcessNetworkResource

func ZeroLengthProcessNetworkReadForTest() (int, error) { return (&processConn{}).Read(nil) }

func ZeroLengthProcessNetworkWriteForTest() (int, error) {
	return (&processConn{}).Write(nil)
}
