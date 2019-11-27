package main

import (
	"encoding/json"
	"fmt"
	"time"
	"strconv"
	"bytes"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// 票据状态
const (
	BillIssued	= "issued"	// 票据通过合同生成
	BillLoanReady	= "loanready"	// 票据申请抵押贷款
	BillMorgaged	= "mortgaged"	// 票据被抵押给金融机构，获得贷款
	BillEndorsed	= "endorsed"	// 担保方承认票据，并为票据担保
	BillRejected	= "rejected"	// 担保方不同意票据担保
	BillAbolished	= "abolished"	// 把票据作废
	BillSplit	= "split"	// 拆分票据
)

//Bill 票据基本结构
type Bill struct {
	ParentID	string	`json:"parent_id"`	//票据来源，生成票据的合同号或被拆分的票据号
	BillID		string	`json:"bill_id"`	//票据号
	Amount		float64	`json:"amount"`		//票据金额
	AmountUnit	string	`json:"amount_unit"`	//金额单位，元或美元等
	IssueDate	int64	`json:"issue_date"`	//票据出票日期
	DueDate		int64	`json:"due_date"`	//票据到期日期
	PyeeName	string	`json:"pyee_name"`	//收款人名称
	PyeeID		string	`json:"pyee_id"`	//收款人身份号
	PyeeAcct	string	`json:"pyee_acct"`	//收款人账户
	Drawee		string	`json:"drawee"`		//还款人账号
	DraweeName	string	`json:"drawee_name"`	//还款人名称
	Issuer		string	`json:"issuer"`		//票据发起人账号
	IssuerName	string	`json:"issuer_name"`	//票据发起人名称
	Bank		string	`json:"bank,omitempty"`		//金融机构账号
	BankeName	string	`json:"bank_name,omitempty"`		//金融机构名称
	Owner		string	`json:"owner"`		//持票人账号
	OwnerName	string	`json:"owner_name"`	//持票人名称
	State		string	`json:"state"`		//票据状态(omitempty,json反序列化显示给客户端时不返回空字段)
}


//BillSplitInfo 票据拆分结构
type BillSplitInfo struct {
	BillID		string		`json:"bill_id"`	//票据号
	OwnerName	string		`json:"owner_name"`	//持票人名称
	Childs		[]BillChild	`json:"child_bills"`	//待拆分的票据
}

//BillChild 待拆分的票据结构
type BillChild struct {
	BillID		string	`json:"bill_id"`	//票据号
	Owner		string	`json:"owner"`		//持票人账号
	OwnerName	string	`json:"owner_name"`	//持票人名称
	Amount		float64	`json:"amount"`		//票据金额
}

// chaincode response结构
type chaincodeRet struct {
	Code int    // 0 success otherwise 1
	Des  string //description
}

//SupplyFinanceBill chaincode基本结构
type SupplyFinanceBill struct {
}

func (bsi BillSplitInfo) SumAmountOfChildBill() float64{
	var sum float64
	sum = 0

	for _, bc := range bsi.Childs {
		sum += bc.Amount
	}

	return sum
}

// response 格式化消息
func getRetByte(code int, des string) []byte {
	var r chaincodeRet
	r.Code = code
	r.Des = des

	b, err := json.Marshal(r)

	if err != nil {
		fmt.Println("marshal Ret failed")
		return nil
	}
	return b
}

// response
func getRetString(code int, des string) string {
	var r chaincodeRet
	r.Code = code
	r.Des = des

	b, err := json.Marshal(r)

	if err != nil {
		fmt.Println("marshal Ret failed")
		return ""
	}
	return string(b[:])
}

// 根据ID查询记录
// args: 0 - id
func (sfb *SupplyFinanceBill) queryByID(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode queryByID args != 1")
		return shim.Error(res)
	}

	id := args[0]
	objBytes, err := stub.GetState(id)
	if  err != nil {
		res := fmt.Sprintf("Chaincode queryByID failed: %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}

	if objBytes == nil {
		return shim.Success([]byte("{}"))
	}

	return shim.Success(objBytes)
}

// 保存票据
// 根据票号取出票据对象
func (sfb *SupplyFinanceBill) getBillObj(stub shim.ChaincodeStubInterface, billNo string) (Bill, bool) {
	var bill Bill
	b, err := stub.GetState(billNo)
	if b == nil {
		return bill, false
	}
	err = json.Unmarshal(b, &bill)
	if err != nil {
		return bill, false
	}
	return bill, true
}

// 保存票据
func (sfb *SupplyFinanceBill) putBill(stub shim.ChaincodeStubInterface, bill Bill) ([]byte, bool) {

	byte, err := json.Marshal(bill)
	if err != nil {
		return nil, false
	}

	err = stub.PutState(bill.BillID, byte)
	if err != nil {
		return nil, false
	}
	return byte, true
}

//Init chaincode基本接口
func (sfb *SupplyFinanceBill) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//Invoke chaincode基本接口
func (sfb *SupplyFinanceBill) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	if function == "issue" {
		return sfb.issue(stub, args)
	} else if function == "endorse" {
		return sfb.endorse(stub, args)
	} else if function == "reject" {
		return sfb.reject(stub, args)
	} else if function == "abolish" {
		return sfb.abolish(stub, args)
	} else if function == "split" {
		return sfb.split(stub, args)
	} else if function == "applyLoan" {
		return sfb.applyLoan(stub, args)
	} else if function == "mortgage" {
		return sfb.mortgage(stub, args)
	} else if function == "queryByID" {
		return sfb.queryByID(stub, args)
	} else if function == "queryBills" {
		return sfb.queryBills(stub, args)
	} else if function == "queryBillsWithPagination" {
		return sfb.queryBillsWithPagination(stub, args)
	} else if function == "queryTXChainForBill" {
		return sfb.queryTXChainForBill(stub, args)
	}

	res := getRetString(1, "Chaincode Unkown method!")

	return shim.Error(res)
}

//issue 票据发布
// args: 0 - {Bill Object}
func (sfb *SupplyFinanceBill) issue(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke issue args != 1")
		return shim.Error(res)
	}

	var bill Bill
	err := json.Unmarshal([]byte(args[0]), &bill)

	if err != nil {
		res := getRetString(1, "Chaincode Invoke issue unmarshal failed")
		return shim.Error(res)
	}

	msg, ok := sfb.issueObj(stub, &bill)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

func (sfb *SupplyFinanceBill) issueObj(stub shim.ChaincodeStubInterface, bill *Bill) (string, bool){
	// 根据票号 查找是否票号已存在
	_, existbl := sfb.getBillObj(stub, bill.BillID)
	if existbl {
		res := fmt.Sprintf("Chaincode Invoke issue failed : the bill has existting, bill NO: %s", bill.BillID)
		return res, false
	}

	// 设置票据的状态
	//bill.State = BillIssued
	bill.State = BillEndorsed

	// 保存票据
	_, ok := sfb.putBill(stub, *bill)
	if !ok {
		return "Chaincode Invoke issue put bill failed", false
	}

	return "invoke issue success", true
}

//endorse 担保票据
//  args: 0 - Bill_No ; 1 - Drawee Name 
func (sfb *SupplyFinanceBill) endorse(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke endorse args count expecting 2")
		return shim.Error(res)
	}
	// 根据票号取得票据
	bill, bl := sfb.getBillObj(stub, args[0])
	if !bl {
		res := getRetString(1, "Chaincode Invoke endorse get bill error")
		return shim.Error(res)
	}

	if bill.DraweeName != args[1] {
		res := getRetString(1, "Chaincode Invoke endorse failed: Endorser is not same with current drawee")
		return shim.Error(res)
	}

	msg, ok := sfb.setBillState(stub, &bill, BillIssued, BillEndorsed)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

func (sfb *SupplyFinanceBill) setBillState(stub shim.ChaincodeStubInterface, bill *Bill, expected_state, set_state string) (string, bool){
	// 检查票据当前状态
	if bill.State != expected_state {
		res := fmt.Sprintf("Chaincode Invoke failed: due to its state, current state: %s", bill.State)
		return res, false
	}

	// 更改票据状态
	bill.State = set_state

	// 保存票据
	_, ok := sfb.putBill(stub, *bill)
	if !ok {
		return "Chaincode Invoke set bill state failed", false
	}

	return "invoke endorse success", true
}

//reject 拒绝担保票据
//  args: 0 - Bill_No ; 1 - Drawee Name 
func (sfb *SupplyFinanceBill) reject(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke reject args count expecting 2")
		return shim.Error(res)
	}

	// 根据票号取得票据
	bill, ok := sfb.getBillObj(stub, args[0])
	if !ok {
		res := getRetString(1, "Chaincode Invoke reject get bill error")
		return shim.Error(res)
	}

	if bill.DraweeName != args[1] {
		res := getRetString(1, "Chaincode Invoke endorse failed: Endorser is not same with current drawee")
		return shim.Error(res)
	}

	msg, ok := sfb.setBillState(stub, &bill, BillIssued, BillRejected)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//mortgage 用票据贷款成功
//  args: 0 - Bill_No ; 1 - Owner ; 2 - Bank; 3 - Bank Name
func (sfb *SupplyFinanceBill) mortgage(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke mortgage args count expecting 2")
		return shim.Error(res)
	}

	// 根据票号取得票据
	bill, b1 := sfb.getBillObj(stub, args[0])
	if !b1 {
		res := getRetString(1, "Chaincode Invoke mortgage get bill error")
		return shim.Error(res)
	}

	if bill.OwnerName != args[1] {
		res := getRetString(1, "Chaincode Invoke mortgage failed: owner is not same with current owner")
		return shim.Error(res)
	}

	msg, ok := sfb.setBillState(stub, &bill, BillLoanReady, BillMorgaged)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//applyLoan 用票据申请贷款
//  args: 0 - Bill_No ; 1 - Owner
func (sfb *SupplyFinanceBill) applyLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke applyLoan args count expecting 2")
		return shim.Error(res)
	}

	// 根据票号取得票据
	bill, b1 := sfb.getBillObj(stub, args[0])
	if !b1 {
		res := getRetString(1, "Chaincode Invoke applyLoan get bill error")
		return shim.Error(res)
	}

	if bill.OwnerName != args[1] {
		res := getRetString(1, "Chaincode Invoke applyLoan failed: owner is not same with current owner")
		return shim.Error(res)
	}

	msg, ok := sfb.setBillState(stub, &bill, BillEndorsed, BillLoanReady)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//abolish 作废票据
//  args: 0 - Bill_No ; 1 - Owner
func (sfb *SupplyFinanceBill) abolish(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke abolish args count expecting 2")
		return shim.Error(res)
	}

	// 根据票号取得票据
	bill, b1 := sfb.getBillObj(stub, args[0])
	if !b1 {
		res := getRetString(1, "Chaincode Invoke abolish get bill error")
		return shim.Error(res)
	}

	if bill.OwnerName != args[1] {
		res := getRetString(1, "Chaincode Invoke abolish failed: owner is not same with current owner")
		return shim.Error(res)
	}

	// 已经抵押贷款和被拆分的票据不运行作废
	if bill.State == BillMorgaged || bill.State == BillSplit {
		res := fmt.Sprintf("Chaincode Invoke abolish failed: The bill can not be abolished due to its state, current state: %s", bill.State)
		res = getRetString(1, res)
		return shim.Error(res)
	}

	// 更改票据状态为拒绝背书担保
	bill.State = BillAbolished

	// 保存票据
	_, ok := sfb.putBill(stub, bill)
	if !ok {
		res := getRetString(1, "Chaincode Invoke abolish put bill failed")
		return shim.Error(res)
	}

	res := getRetByte(0, "invoke accept success")

	return shim.Success(res)
}

/*split 拆分票据
**  args: 0 - {split object json}
**  sample:
**  {
**	"bill_id":"123789",
**	"owner_name":"国信泰一",
**  	"child_bills":
**  	[
**		{
**			"bill_id":"0001",
**			"owner":"gt1",
**			"owner_name":"国泰公司1",
**			"amount":70000
**		},
**		{
**			"bill_id":"00002",
**			"owner":"gt2",
**			"owner_name":"国泰公司2",
**			"amount":90000
**		}
**	]
**  }
*/
func (sfb *SupplyFinanceBill) split(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke split args count expecting 1")
		return shim.Error(res)
	}

	var bsi BillSplitInfo
	err := json.Unmarshal([]byte(args[0]), &bsi)

	if err != nil {
		res := getRetString(1, "Chaincode Invoke split unmarshal failed")
		return shim.Error(res)
	}

	b, exist := sfb.getBillObj(stub, bsi.BillID)
	if !exist {
		res := fmt.Sprintf("Chaincode Invoke[split] failed : the bill is not existing, bill NO: %s", bsi.BillID)
		res = getRetString(1, res)
		return shim.Error(res)
	}

	if b.OwnerName != bsi.OwnerName {
		res := getRetString(1, "Chaincode Invoke abolish failed: owner is not same with current owner")
		return shim.Error(res)
	}

	// 只有通过背书担保的票据才能拆分
	if b.State != BillEndorsed{
		res := fmt.Sprintf("Chaincode Invoke split failed: The bill can not be split due to its state, current state: %s", b.State)
		res = getRetString(1, res)
		return shim.Error(res)
	}

	sumAmount := bsi.SumAmountOfChildBill()
	if b.Amount != sumAmount {
		res := fmt.Sprintf("Chaincode Invoke split failed: The total amount of all child bills is not equal the parent's amount")
		res = getRetString(1, res)
		return shim.Error(res)
	}

	if len(bsi.Childs) < 2 {
		res := getRetString(1, "Chaincode Invoke split failed: at least 2 Sub-Bills are required")
		return shim.Error(res)
	}

	b_child := b
	b_child.ParentID = bsi.BillID
	for _, bc := range bsi.Childs {
		b_child.BillID= bc.BillID
		b_child.Owner = bc.Owner
		b_child.OwnerName = bc.OwnerName
		b_child.Amount = bc.Amount

		msg, ok := sfb.issueObj(stub, &b_child)
		if !ok {
			res := getRetString(1, msg)
			return shim.Error(res)
		}
	}

	b.State = BillSplit
	_, ok := sfb.putBill(stub, b)
	if !ok {
		res := getRetString(1, "Chaincode Invoke split save bill failed")
		return shim.Error(res)
	}

	res := getRetByte(0, "invoke endorse success")
	return shim.Success(res)

}

//queryMarblesWithPagination 分页查询票据发起人、持有人、还款人的所有票据
//  0 - Issuer|Drawee|Owner ; 1 - count of page ; 2 - pagination bookmark
func (t *SupplyFinanceBill) queryBillsWithPagination(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		return shim.Error("Chaincode query[queryMarblesWithPagination] failed: argument expecting 3")
	}

	queryString := args[0]
	//return type of ParseInt is int64
	pageSize, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return shim.Error(err.Error())
	}
	bookmark := args[2]

	queryResults, err := getQueryResultForQueryStringWithPagination(stub, queryString, int32(pageSize), bookmark)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func getQueryResultForQueryStringWithPagination(stub shim.ChaincodeStubInterface, queryString string, pageSize int32, bookmark string) ([]byte, error) {
	resultsIterator, responseMetadata, err := stub.GetQueryResultWithPagination(queryString, pageSize, bookmark)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	bf_data, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	bf_meta := constructPaginationMetadataToQueryResults(responseMetadata)

	bf := constructJsonArray(bf_meta, bf_data)

	return bf.Bytes(), nil
}

func constructPaginationMetadataToQueryResults(responseMetadata *pb.QueryResponseMetadata) *bytes.Buffer {
	var buffer bytes.Buffer

	buffer.WriteString("{\"ResponseMetadata\":{\"RecordsCount\":")
	buffer.WriteString("\"")
	buffer.WriteString(fmt.Sprintf("%v", responseMetadata.FetchedRecordsCount))
	buffer.WriteString("\"")
	buffer.WriteString(", \"Bookmark\":")
	buffer.WriteString("\"")
	buffer.WriteString(responseMetadata.Bookmark)
	buffer.WriteString("\"}}")

	return &buffer
}

//queryBills 查询票据发起人、持有人、还款人的所有票据
//  0 - Issuer|Drawee|Owner ;
func (t *SupplyFinanceBill) queryBills(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Chaincode query[queryBills] failed: argument expecting 1")
	}

	queryString := args[0]

	queryResults, err := getQueryResultForQueryString(stub, queryString)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(queryResults)
}

func getQueryResultForQueryString(stub shim.ChaincodeStubInterface, queryString string) ([]byte, error) {
	resultsIterator, err := stub.GetQueryResult(queryString)
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	buffer, err := constructQueryResponseFromIterator(resultsIterator)
	if err != nil {
		return nil, err
	}

	bf := constructJsonArray(buffer)

	return bf.Bytes(), nil
}

func constructJsonArray(bufs... *bytes.Buffer) *bytes.Buffer {
	var buffer bytes.Buffer

	buffer.WriteString("[")
	for i, b := range bufs {
		if i != 0 {
			buffer.WriteString(",")
		}
		buffer.Write(b.Bytes())
	}
	buffer.WriteString("]")

	return &buffer
}

func constructQueryResponseFromIterator(resultsIterator shim.StateQueryIteratorInterface) (*bytes.Buffer, error) {
	var buffer bytes.Buffer

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"Key\":")
		buffer.WriteString("\"")
		buffer.WriteString(queryResponse.Key)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Record\":")
		// Record is a JSON object, so we write as-is
		buffer.WriteString(string(queryResponse.Value))
		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}

	return &buffer, nil
}

//queryTXChainForBill 根据票号取得票据交易链
//  0 - Bill_No ;
func (t *SupplyFinanceBill) queryTXChainForBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error("Chaincode query[queryTXChainForBill] failed: argument expecting 1")
	}

	billID := args[0]

	resultsIterator, err := stub.GetHistoryForKey(billID)
	if err != nil {
		return shim.Error(err.Error())
	}
	defer resultsIterator.Close()

	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return shim.Error(err.Error())
		}
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	return shim.Success(buffer.Bytes())
}

//queryByBillNo 根据票号取得票据 以及该票据背书历史
//  0 - Bill_No ;
func (sfb *SupplyFinanceBill) queryByBillNo(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode queryByBillNo args!=1")
		return shim.Error(res)
	}
	// 取得该票据
	_, ok := sfb.getBillObj(stub, args[0])
	if !ok {
		res := getRetString(1, "Chaincode queryByBillNo get bill error")
		return shim.Error(res)
	}

	return shim.Success(nil)
}

func main() {
	if err := shim.Start(new(SupplyFinanceBill)); err != nil {
		fmt.Printf("Error starting SupplyFinanceBill: %s", err)
	}
}
