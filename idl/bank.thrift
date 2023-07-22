namespace go api

struct BankNameReq {
    1: string Name
}

struct BankNameResp {
    1: string RespBody
}

service bank {
    BankNameResp name(1: BankNameReq request) (api.get="/bank/name")
}