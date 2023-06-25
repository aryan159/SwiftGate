namespace go api

struct BankNameReq {
    1: string Name (api.query="name")
}

struct BankNameResp {
    1: string RespBody
}

service BankService {
    HelloResp HelloMethod(1: HelloReq request) (api.get="/HelloService/HelloMethod")
    EchoResp echo(1: EchoReq request) (api.get="/HelloService/echo")
}