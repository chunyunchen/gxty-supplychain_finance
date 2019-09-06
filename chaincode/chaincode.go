package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// logger
var chaincodeLogger = flogging.MustGetLogger("Chaincode")

// 票据状态
const (
	BillInfoStateNewPublish   = "NewPublish"
	BillInfoStateEndrWaitSign = "EndrWaitSign"
	BillInfoStateEndrSigned   = "EndrSigned"
	BillInfoStateEndrReject   = "EndrReject"
	BillInfoStateAbolish      = "Abolish"
	BillInfoStateSplit        = "Split"
)

//BillPrefix 票据key的前缀
const BillPrefix = "Bill_"

//IndexName search表的映射名
const IndexName = "holderName~billNo"

//HolderIDDayTimeBillTypeBillNoIndexName 批量查询匹配
const HolderIDDayTimeBillTypeBillNoIndexName = "holderId~dayTime-billType-billNo"

//Bill 票据基本结构
type Bill struct {
	BillInfoID         string        `json:"BillInfoID"`         //票据号码
	BillInfoAmt        string        `json:"BillInfoAmt"`        //票据金额
	BillInfoType       string        `json:"BillInfoType"`       //票据类型
	BillInfoIsseDate   string        `json:"BillInfoIsseDate"`   //票据出票日期
	BillInfoDueDate    string        `json:"BillInfoDueDate"`    //票据到期日期
	DrwrCmID           string        `json:"DrwrCmID"`           //出票人证件号码
	DrwrAcct           string        `json:"DrwrAcct"`           //出票人名称
	AccptrCmID         string        `json:"AccptrCmID"`         //承兑人证件号码
	AccptrAcct         string        `json:"AccptrAcct"`         //承兑人名称
	PyeeCmID           string        `json:"PyeeCmID"`           //收款人证件号码
	PyeeAcct           string        `json:"PyeeAcct"`           //收款人名称
	HodrCmID           string        `json:"HodrCmID"`           //持票人证件号码
	HodrAcct           string        `json:"HodrAcct"`           //持票人名称
	WaitEndorserCmID   string        `json:"WaitEndorserCmID"`   //待背书人证件号码
	WaitEndorserAcct   string        `json:"WaitEndorserAcct"`   //待背书人名称
	RejectEndorserCmID string        `json:"RejectEndorserCmID"` //拒绝背书人证件号码
	RejectEndorserAcct string        `json:"RejectEndorserAcct"` //拒绝背书人名称
	State              string        `json:"State"`              //票据状态
	History            []HistoryItem `json:"History"`            //背书历史
}

//HistoryItem 背书历史item结构
type HistoryItem struct {
	TxID string `json:"txID"`
	Bill Bill   `json:"bill"`
}

// chaincode response结构
type chaincodeRet struct {
	Code int    // 0 success otherwise 1
	Des  string //description
}

//BillChaincode chaincode基本结构
type BillChaincode struct {
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
	chaincodeLogger.Infof("%s", string(b[:]))
	return string(b[:])
}

// 根据票号取出票据
func (a *BillChaincode) getBill(stub shim.ChaincodeStubInterface, billNo string) (Bill, bool) {
	var bill Bill
	key := BillPrefix + billNo
	b, err := stub.GetState(key)
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
func (a *BillChaincode) putBill(stub shim.ChaincodeStubInterface, bill Bill) ([]byte, bool) {

	byte, err := json.Marshal(bill)
	if err != nil {
		return nil, false
	}

	err = stub.PutState(BillPrefix+bill.BillInfoID, byte)
	if err != nil {
		return nil, false
	}
	return byte, true
}

//Init chaincode基本接口
func (a *BillChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//Invoke chaincode基本接口
func (a *BillChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	chaincodeLogger.Info("%s%s", "Chaincode function=", function)
	chaincodeLogger.Info("%s%s", "Chaincode args=", args)

	// invoke
	if function == "issue" {
		return a.issue(stub, args)
	} else if function == "endorse" {
		return a.endorse(stub, args)
	} else if function == "accept" {
		return a.accept(stub, args)
	} else if function == "reject" {
		return a.reject(stub, args)
	} else if function == "abolish" {
		return a.abolish(stub, args)
	} else if function == "split" {
		return a.split(stub, args)
	}

	// query
	if function == "queryMyBill" {
		return a.queryMyBill(stub, args)
	} else if function == "queryByBillNo" {
		return a.queryByBillNo(stub, args)
	} else if function == "queryMyWaitBill" {
		return a.queryMyWaitBill(stub, args)
	}

	res := getRetString(1, "Chaincode Unkown method!")
	chaincodeLogger.Info("%s", res)
	chaincodeLogger.Infof("%s", res)
	return shim.Error(res)
}

//issue 票据发布
// args: 0 - {Bill Object}
func (a *BillChaincode) issue(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke issue args!=1")
		return shim.Error(res)
	}

	var bill Bill
	err := json.Unmarshal([]byte(args[0]), &bill)
	if err != nil {
		res := getRetString(1, "Chaincode Invoke issue unmarshal failed")
		return shim.Error(res)
	}
	// TODO 根据票号 查找是否票号已存在
	// TODO 如stat中已有同号票据 返回error message
	_, existbl := a.getBill(stub, bill.BillInfoID)
	if existbl {
		res := getRetString(1, "Chaincode Invoke issue failed : the billNo has exist ")
		return shim.Error(res)
	}

	timestamp, err := stub.GetTxTimestamp()
	if err != nil {
		res := getRetString(1, "Chaincode Invoke issue failed :get time stamp failed ")
		return shim.Error(res)
	}
	chaincodeLogger.Info("%s", timestamp)

	var dayTime = time.Now().Format("2009-10-10")

	resultIterator, err := stub.GetStateByPartialCompositeKey(HolderIDDayTimeBillTypeBillNoIndexName, []string{bill.HodrCmID, dayTime, bill.BillInfoType})
	if err != nil {
		res := getRetString(1, "Chaincode Invoke issue get bill list error")
		return shim.Error(res)
	}
	defer resultIterator.Close()

	var count = 0
	for resultIterator.HasNext() {
		_, _ = resultIterator.Next()

		count++

		if count >= 5 {
			res := getRetString(1, "Chaincode Invoke issue The bill holder has more than 5 bills on the same day by the same type")
			return shim.Error(res)
		}
	}

	holderIDDayTimeBillNoIndexKey, err := stub.CreateCompositeKey(HolderIDDayTimeBillTypeBillNoIndexName, []string{bill.HodrCmID, dayTime, bill.BillInfoType, bill.BillInfoID})
	if err != nil {
		res := getRetString(1, "Chaincode Invoke issue put search table failed")
		return shim.Error(res)
	}
	stub.PutState(holderIDDayTimeBillNoIndexKey, []byte(time.Now().Format("2017-11-20 12:56:56")))

	// 更改票据信息和状态并保存票据:票据状态设为新发布
	bill.State = BillInfoStateNewPublish
	// 保存票据
	_, bl := a.putBill(stub, bill)
	if !bl {
		res := getRetString(1, "Chaincode Invoke issue put bill failed")
		return shim.Error(res)
	}
	// 以持票人ID和票号构造复合key 向search表中保存 value为空即可 以便持票人批量查询
	holderNameBillNoIndexKey, err := stub.CreateCompositeKey(IndexName, []string{bill.HodrCmID, bill.BillInfoID})
	if err != nil {
		res := getRetString(1, "Chaincode Invoke issue put search table failed")
		return shim.Error(res)
	}
	stub.PutState(holderNameBillNoIndexKey, []byte{0x00})

	res := getRetByte(0, "invoke issue success")
	return shim.Success(res)
}

//endorse 背书请求
//  args: 0 - Bill_No ; 1 - Endorser CmId ; 2 - Endorser Acct
func (a *BillChaincode) endorse(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		res := getRetString(1, "Chaincode Invoke endorse args<3")
		return shim.Error(res)
	}
	// 根据票号取得票据
	bill, bl := a.getBill(stub, args[0])
	if !bl {
		res := getRetString(1, "Chaincode Invoke endorse get bill error")
		return shim.Error(res)
	}

	if bill.HodrCmID == args[1] {
		res := getRetString(1, "Chaincode Invoke endorse failed: Endorser should not be same with current Holder")
		return shim.Error(res)
	}

	// 查询背书人是否以前持有过该数据
	// 取得背书历史: 通过fabric api取得该票据的变更历史
	resultsIterator, err := stub.GetHistoryForKey(BillPrefix + args[0])
	if err != nil {
		res := getRetString(1, "Chaincode queryByBillNo GetHistoryForKey error")
		return shim.Error(res)
	}
	defer resultsIterator.Close()

	var hisBill Bill
	for resultsIterator.HasNext() {
		historyData, err := resultsIterator.Next()
		if err != nil {
			res := getRetString(1, "Chaincode queryByBillNo resultsIterator.Next() error")
			return shim.Error(res)
		}

		var hodlerNameList []string
		json.Unmarshal(historyData.Value, &hisBill) //un stringify it aka JSON.parse()
		if historyData.Value == nil {               //bill has been deleted
			var emptyBill Bill
			hisBill = emptyBill //copy nil marble
		} else {
			json.Unmarshal(historyData.Value, &hisBill) //un stringify it aka JSON.parse()
			// hisBill = hisBill                           //copy bill over
		}
		hodlerNameList = append(hodlerNameList, hisBill.HodrCmID) //add this tx to the list

		if hisBill.HodrCmID == args[1] {
			res := getRetString(1, "Chaincode Invoke endorse failed: Endorser should not be the bill history holder")
			return shim.Error(res)
		}
	}

	// 更改票据信息和状态并保存票据: 添加待背书人信息,重制已拒绝背书人, 票据状态改为待背书
	bill.WaitEndorserCmID = args[1]
	bill.WaitEndorserAcct = args[2]
	bill.RejectEndorserCmID = ""
	bill.RejectEndorserAcct = ""
	bill.State = BillInfoStateEndrWaitSign
	// 保存票据
	_, bl = a.putBill(stub, bill)
	if !bl {
		res := getRetString(1, "Chaincode Invoke endorse put bill failed")
		return shim.Error(res)
	}
	// 以待背书人ID和票号构造复合key 向search表中保存 value为空即可 以便待背书人批量查询
	holderNameBillNoIndexKey, err := stub.CreateCompositeKey(IndexName, []string{bill.WaitEndorserCmID, bill.BillInfoID})
	if err != nil {
		res := getRetString(1, "Chaincode Invoke endorse put search table failed")
		return shim.Error(res)
	}
	stub.PutState(holderNameBillNoIndexKey, []byte{0x00})

	res := getRetByte(0, "invoke endorse success")
	return shim.Success(res)
}

//accept 背书人接受背书
// args: 0 - Bill_No ; 1 - Endorser CmId ; 2 - Endorser Acct
func (a *BillChaincode) accept(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		res := getRetString(1, "Chaincode Invoke accept args<3")
		return shim.Error(res)
	}
	// 根据票号取得票据
	bill, bl := a.getBill(stub, args[0])
	if !bl {
		res := getRetString(1, "Chaincode Invoke accept get bill error")
		return shim.Error(res)
	}

	// 更改票据信息和状态并保存票据: 将前手持票人改为背书人,重置待背书人,票据状态改为背书签收
	bill.HodrCmID = args[1]
	bill.HodrAcct = args[2]
	bill.WaitEndorserCmID = ""
	bill.WaitEndorserAcct = ""
	bill.State = BillInfoStateEndrSigned
	// 保存票据
	_, bl = a.putBill(stub, bill)
	if !bl {
		res := getRetString(1, "Chaincode Invoke accept put bill failed")
		return shim.Error(res)
	}

	res := getRetByte(0, "invoke accept success")
	return shim.Success(res)
}

//reject 背书人拒绝背书
// args: 0 - Bill_No ; 1 - Endorser CmId ; 2 - Endorser Acct
func (a *BillChaincode) reject(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) < 3 {
		res := getRetString(1, "Chaincode Invoke reject args<3")
		return shim.Error(res)
	}
	// 根据票号取得票据
	bill, bl := a.getBill(stub, args[0])
	if !bl {
		res := getRetString(1, "Chaincode Invoke reject get bill error")
		return shim.Error(res)
	}

	// 维护search表: 以当前背书人ID和票号构造复合key 从search表中删除该key 以便当前背书人无法再查到该票据
	holderNameBillNoIndexKey, err := stub.CreateCompositeKey(IndexName, []string{args[1], bill.BillInfoID})
	if err != nil {
		res := getRetString(1, "Chaincode Invoke reject put search table failed")
		return shim.Error(res)
	}
	stub.DelState(holderNameBillNoIndexKey)

	// 更改票据信息和状态并保存票据: 将拒绝背书人改为当前背书人，重置待背书人,票据状态改为背书拒绝
	bill.WaitEndorserCmID = ""
	bill.WaitEndorserAcct = ""
	bill.RejectEndorserCmID = args[1]
	bill.RejectEndorserAcct = args[2]
	bill.State = BillInfoStateEndrReject
	// 保存票据
	_, bl = a.putBill(stub, bill)
	if !bl {
		res := getRetString(1, "Chaincode Invoke reject put bill failed")
		return shim.Error(res)
	}

	res := getRetByte(0, "invoke accept success")
	return shim.Success(res)
}

func (a *BillChaincode) abolish(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// 根据票号取得票据
	bill, b1 := a.getBill(stub, args[0])
	if !b1 {
		res := getRetString(1, "Chaincode Invoke reject get bill error")
		return shim.Error(res)
	}

	// 维护search表: 以当前背书人ID和票号构造复合key 从search表中删除该key 以便当前背书人无法再查到该票据
	holderNameBillNoIndexKey, err := stub.CreateCompositeKey(IndexName, []string{args[1], bill.BillInfoID})
	if err != nil {
		res := getRetString(1, "Chaincode Invoke reject put search table failed")
		return shim.Error(res)
	}
	stub.DelState(holderNameBillNoIndexKey)

	// 更改票据信息和状态并保存票据: 票据状态改为废弃状态
	bill.WaitEndorserCmID = ""
	bill.WaitEndorserAcct = ""
	bill.RejectEndorserCmID = ""
	bill.RejectEndorserAcct = ""
	bill.State = BillInfoStateAbolish
	// 保存票据
	_, bl := a.putBill(stub, bill)
	if !bl {
		res := getRetString(1, "Chaincode Invoke abolish put bill failed")
		return shim.Error(res)
	}

	res := getRetByte(0, "invoke accept success")

	return shim.Success(res)
}

func (a *BillChaincode) split(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	var bill0 Bill
	err := json.Unmarshal([]byte(args[0]), &bill0)
	if err != nil {
		res := getRetString(1, "Chaincode Invoke split unmarshal failed")
		return shim.Error(res)
	}
	var i int
	// var bill []Bill
	// bill=make([]Bill,len(args)-1)
	for i = 1; i <= len(args)-1; i++ {
		// err:=json.Unmarshal(args[1],bill[i])
		// if err != nil {
		// 	res := getRetString(1, "Chaincode Invoke split unmarshal failed")
		// 	return shim.Error(res+strconv.Itoa(i))
		// }
		str := make([]string, 0)
		str = append(str, args[i])
		res := a.issue(stub, str)
		if res.Status == 500 {
			return res
		}
	}
	b0 := make([]string, 0)
	b0 = append(b0, args[0])
	result := a.abolish(stub, b0)
	return result
}

//queryMyBill 查询我的票据:根据持票人编号 批量查询票据
//  0 - Holder CmId ;
func (a *BillChaincode) queryMyBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode queryMyBill args!=1")
		return shim.Error(res)
	}
	// 以持票人ID从search表中批量查询所持有的票号
	billsIterator, err := stub.GetStateByPartialCompositeKey(IndexName, []string{args[0]})
	if err != nil {
		res := getRetString(1, "Chaincode queryMyBill get bill list error")
		return shim.Error(res)
	}
	defer billsIterator.Close()

	var billList = []Bill{}

	for billsIterator.HasNext() {
		kv, _ := billsIterator.Next()
		// 取得持票人名下的票号
		_, compositeKeyParts, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			res := getRetString(1, "Chaincode queryMyBill SplitCompositeKey error")
			return shim.Error(res)
		}
		// 根据票号取得票据
		bill, bl := a.getBill(stub, compositeKeyParts[1])
		if !bl {
			res := getRetString(1, "Chaincode queryMyBill get bill error")
			return shim.Error(res)
		}
		billList = append(billList, bill)
	}
	// 取得并返回票据数组
	b, err := json.Marshal(billList)
	if err != nil {
		res := getRetString(1, "Chaincode Marshal queryMyBill billList error")
		return shim.Error(res)
	}
	return shim.Success(b)
}

//queryByBillNo 根据票号取得票据 以及该票据背书历史
//  0 - Bill_No ;
func (a *BillChaincode) queryByBillNo(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode queryByBillNo args!=1")
		return shim.Error(res)
	}
	// 取得该票据
	bill, bl := a.getBill(stub, args[0])
	if !bl {
		res := getRetString(1, "Chaincode queryByBillNo get bill error")
		return shim.Error(res)
	}

	// 取得背书历史: 通过fabric api取得该票据的变更历史
	resultsIterator, err := stub.GetHistoryForKey(BillPrefix + args[0])
	if err != nil {
		res := getRetString(1, "Chaincode queryByBillNo GetHistoryForKey error")
		return shim.Error(res)
	}
	defer resultsIterator.Close()

	var history []HistoryItem
	var hisBill Bill
	for resultsIterator.HasNext() {
		historyData, err := resultsIterator.Next()
		if err != nil {
			res := getRetString(1, "Chaincode queryByBillNo resultsIterator.Next() error")
			return shim.Error(res)
		}

		var hisItem HistoryItem
		hisItem.TxID = historyData.TxId             //copy transaction id over
		json.Unmarshal(historyData.Value, &hisBill) //un stringify it aka JSON.parse()
		if historyData.Value == nil {               //bill has been deleted
			var emptyBill Bill
			hisItem.Bill = emptyBill //copy nil marble
		} else {
			json.Unmarshal(historyData.Value, &hisBill) //un stringify it aka JSON.parse()
			hisItem.Bill = hisBill                      //copy bill over
		}
		history = append(history, hisItem) //add this tx to the list
	}
	// 将背书历史做为票据的一个属性 一同返回
	bill.History = history

	b, err := json.Marshal(bill)
	if err != nil {
		res := getRetString(1, "Chaincode Marshal queryByBillNo billList error")
		return shim.Error(res)
	}
	return shim.Success(b)
}

//queryMyWaitBill 查询我的待背书票据: 根据背书人编号 批量查询票据
//  0 - Endorser CmId ;
func (a *BillChaincode) queryMyWaitBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode queryMyWaitBill args!=1")
		return shim.Error(res)
	}
	// 以背书人ID从search表中批量查询所持有的票号
	billsIterator, err := stub.GetStateByPartialCompositeKey(IndexName, []string{args[0]})
	if err != nil {
		res := getRetString(1, "Chaincode queryMyWaitBill GetStateByPartialCompositeKey error")
		return shim.Error(res)
	}
	defer billsIterator.Close()

	var billList = []Bill{}

	for billsIterator.HasNext() {
		kv, _ := billsIterator.Next()
		// 从search表中批量查询与背书人有关的票号
		_, compositeKeyParts, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			res := getRetString(1, "Chaincode queryMyWaitBill SplitCompositeKey error")
			return shim.Error(res)
		}
		// 根据票号取得票据
		bill, bl := a.getBill(stub, compositeKeyParts[1])
		if !bl {
			res := getRetString(1, "Chaincode queryMyWaitBill get bill error")
			return shim.Error(res)
		}
		// 取得状态为待背书的票据 并且待背书人是当前背书人
		if bill.State == BillInfoStateEndrWaitSign && bill.WaitEndorserCmID == args[0] {
			billList = append(billList, bill)
		}
	}
	// 取得并返回票据数组
	b, err := json.Marshal(billList)
	if err != nil {
		res := getRetString(1, "Chaincode Marshal queryMyWaitBill billList error")
		return shim.Error(res)
	}
	return shim.Success(b)
}

func main() {
	if err := shim.Start(new(BillChaincode)); err != nil {
		fmt.Printf("Error starting BillChaincode: %s", err)
	}
}
