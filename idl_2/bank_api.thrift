namespace go api

struct BankNameReq {
    1: string Name
}

struct BankNameResp {
    1: string RespBody
}

service BankService {
    BankNameResp GetNameMethod(1: BankNameReq request) (api.get="/bank/name")
}