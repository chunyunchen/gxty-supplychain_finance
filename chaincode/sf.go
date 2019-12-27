package main

import (
	"encoding/json"
	"fmt"
	"time"
	"strconv"
	"bytes"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
//	ms "github.com/chaincode/bcsf/mapstructure"
)

// 票据和贷款状态
const (
	BillIssued	= "issued"	// 票据通过合同生成
	BillLoanReady	= "loanready"	// 票据申请抵押贷款
	BillMorgaged	= "mortgaged"	// 票据被抵押给金融机构，获得贷款
	BillAbolished	= "abolished"	// 把票据作废
	BillSplit	= "split"	// 拆分票据
	BillRedeemed	= "redeemed"	// 已还款，票据赎回
	LoanGurantee	= "untrusted"	// 申请信用企业为贷款提供担保
	LoanApplied	= "applied"	// 贷款已经申请，等待银行审批
	LoanRefused	= "refused"	// 银行拒绝贷款
	LoanApproved	= "approved"	// 银行同意贷款
	LoanLoaned	= "loaned"	// 银行放款
	LoanRepaid	= "repaid"	// 贷款已还款
	ContractUploaded= "uploaded"	// 合同已经上传
	Endorsed	= "endorsed"	// 同意为合同或票据或贷款担保
	Rejected	= "rejected"	// 拒绝为合同或票据或贷款担保
)

// 原始票据最大拆分深度/次数
const SplitThreshold = 1

//Loan 贷款信息基本结构
type Loan struct {
	LoanID		string	`json:"loan_id"`	//贷款编号
	BillID		string	`json:"ln_bill_id"`	//票据号
	Amount		float64	`json:"ln_amount"`	//贷款金额
	AmountUnit	string	`json:"ln_amount_unit"`	//金额单位，元或美元等
	BankRate	float64	`json:"bank_rate"`	//贷款利率
	BankInterest	float64	`json:"bank_interest"`	//贷款利息
	PyeeAcct	string	`json:"ln_pyee_acct"`	//收款人账户
	Owner		string	`json:"ln_owner"`	//贷款人系统账号
	OwnerName	string	`json:"ln_owner_name"`	//贷款人名称
	State		string	`json:"ln_state"`		//贷款状态
	Guarantor	string	`json:"guarantor,omitempty"`		//担保方/还款人系统账号
	GuarantorName	string	`json:"guarantor_name,omitempty"`	//担保方/还款人名称
	Bank		string	`json:"ln_bank,omitempty"`		//金融机构系统账号
	BankName	string	`json:"ln_bank_name,omitempty"`		//金融机构名称
	RepaymentDate   int64	`json:"repayment_date"`			//还款时间
	RefuseReason	string	`json:"refused_reason,omitempty"`	//拒绝贷款原因
	ApplyDate	int64	`json:"apply_date"`	//贷款申请时间
}

func (ln Loan) ValidateGuarantorName(expectedV string) bool {
	if ln.GuarantorName == expectedV {
		return true
	}
	
	return false
}

func (ln Loan) ValidateBankName(expectedV string) bool {
	if ln.BankName == expectedV {
		return true
	}
	
	return false
}

func (ln Loan) ValidateOwnerName(expectedV string) bool {
	if ln.OwnerName == expectedV {
		return true
	}
	
	return false
}

func (ln Loan) ValidateState(expectedV string) bool {
	if ln.State == expectedV {
		return true
	}
	
	return false
}

//LoanResultArg 审核贷款结果参数结构
type LoanResultArg struct {
	LoanID		string	`json:"loan_id"`	//贷款编号
	OwnerName	string	`json:"ln_owner_name"`	//贷款人名称
	Bank		string	`json:"ln_bank"`	//金融机构系统账号
	BankName	string	`json:"ln_bank_name"`	//金融机构名称
	RefuseReason	string	`json:"refused_reason"`	//拒绝贷款原因
}

//LoanRepayment 还款信息结构
type LoanRepayment struct {
	LoanID		string	`json:"lr_loan_id"`	//贷款编号
	IsPrepayment	bool	`json:"prepayment"` //是否提前还款
	makeLoanDate	int64	`json:"make_loan_date,omitempty"`	//贷款放款时间
	ActualRepaymentDate   int64	`json:"actual_repayment_date,omitempty"`	//实际还款时间
	ActualAmount		float64	`json:"actual_lr_amount,omitempty"`	//贷款实际还款金额
	AmountUnit	string	`json:"ln_amount_unit,omitempty"`	//金额单位，元或美元等
	ActualBankRate	float64	`json:"actual_bank_rate,omitempty"`	//还款时的贷款利率
	ActualBankInterest	float64	`json:"actual_bank_interest,omitempty"`	//还款时的贷款利息
}

//LoanRepaymentArg 还贷信息参数
type LoanRepaymentArg struct {
	LoanID		string	`json:"lr_loan_id"`	//贷款编号
	BankName	string	`json:"lr_bank_name,omitempty"`	//金融机构名称
	ActualRepaymentDate   int64	`json:"actual_repayment_date,omitempty"`	//实际还款时间
	ActualAmount		float64	`json:"actual_ln_amount,omitempty"`	//贷款实际还款金额
	AmountUnit	string	`json:"lr_amount_unit,omitempty"`	//金额单位，元或美元等
	ActualBankRate	float64	`json:"actual_bank_rate,omitempty"`	//还款时的贷款利率
	ActualBankInterest	float64	`json:"actual_bank_interest,omitempty"`	//还款时的贷款利息
}

//Contract 合同基本结构
type Contract struct {
	ContractID	string	`json:"contract_id"`	//合同号
	HashID		string	`json:"hash_id"`	//合同内容的hash值
	BillHashID	string	`json:"bill_hash_id,omitempty"`	//票据内容的hash值，线下上传票据文件时通过文件内容计算
	Amount		float64	`json:"ct_amount"`		//合同金额
	AmountUnit	string	`json:"ct_amount_unit"`	//金额单位，元或美元等
	IssueDate	int64	`json:"ct_issue_date"`	//开始日期
	DueDate		int64	`json:"ct_due_date"`	//到期日期
	PyeeName	string	`json:"ct_pyee_name"`	//收款人名称
	PyeeID		string	`json:"ct_pyee_id"`	//收款人身份号
	PyeeAcct	string	`json:"ct_pyee_acct"`	//收款人账户
	Drawee		string	`json:"ct_drawee"`		//还款人系统账号
	DraweeName	string	`json:"ct_drawee_name"`	//还款人名称
	Issuer		string	`json:"ct_issuer"`		//发起人系统账号
	IssuerName	string	`json:"ct_issuer_name"`	//发起人名称
	Owner		string	`json:"ct_owner"`		//持有人系统账号
	OwnerName	string	`json:"ct_owner_name"`	//持有人名称
	State		string	`json:"ct_state"`		//合同状态
	RefuseReason	string	`json:"refused_reason,omitempty"`	//拒绝担保合同原因
}

func (ct Contract) ValidateDraweeName(expectedV string) bool {
	if ct.DraweeName == expectedV {
		return true
	}
	
	return false
}

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
	Drawee		string	`json:"drawee"`		//还款人系统账号
	DraweeName	string	`json:"drawee_name"`	//还款人名称
	Issuer		string	`json:"issuer"`		//票据发起人系统账号
	IssuerName	string	`json:"issuer_name"`	//票据发起人名称
	Owner		string	`json:"owner"`		//持票人系统账号
	OwnerName	string	`json:"owner_name"`	//持票人名称
	State		string	`json:"state"`		//票据状态(omitempty,json反序列化显示给客户端时不返回空字段)
	SplitCount	int32	`json:"split_count"`    //控制原始票据拆分次数，该值表示当前票据是通过几次拆分而生成的
	Transferred	bool	`json:"transferred"`    //票据是否流转过，false - 没流转过的票据，true - 流转过的票据
}

const(
	THOUSAND int64 = 1000
    TEN_MILLION int64 = 1000000
)

func getSecAndNSec(milliSec int64) (int64, int64){
	Sec := milliSec / THOUSAND
	NSec := milliSec % THOUSAND * TEN_MILLION
	return Sec,NSec
}

func afterNowDate(t time.Time) bool {
	now := time.Now()
	
	if t.Year() > now.Year() {
		return true
	} else if t.Year() == now.Year() {
		if t.Month() > now.Month() {
			return true
		} else if t.Month() == now.Month() {
			if t.Day() >= now.Day() {
				return true
			}
		}
	}
	
	return false
}

func (bl Bill) ValidateDueDate() bool {
	// 检查票据的到期时间是否已过期
	Sec, NSec := getSecAndNSec(bl.DueDate)
    t := time.Unix(Sec, NSec)

	if afterNowDate(t) {
		return true
	}
	
 	return false
}
 
func (bl Bill) ValidateSplitCount(expectedV int32) bool {
	if bl.SplitCount < expectedV {
		return true
	}
	
	return false
}

func (bl Bill) ValidateDraweeName(expectedV string) bool {
	if bl.DraweeName == expectedV {
		return true
	}
	
	return false
}

func (bl Bill) ValidateOwner(expectedV string) bool {
	if bl.Owner == expectedV {
		return true
	}
	
	return false
}

func (bl Bill) ValidateOwnerName(expectedV string) bool {
	if bl.OwnerName == expectedV {
		return true
	}
	
	return false
}

func (bl Bill) ValidateState(expectedV string) bool {
	if bl.State == expectedV {
		return true
	}
	
	return false
}

//TransferredBill 企业流转出去的票据集合结构
type TransferredBill struct {
	Owner	string		`json:"tb_owner"`	//企业系统账号
	Bills	[]string	`json:"bills"`	//票据号集合
}

//BillTransfer 票据流转信息结构
type BillTransfer struct {
	BillID		string	//票据编号
	Count		int		//票据已经流转的总次数
	Transfers	map[int]TransferInfoArg  //key：流转次数序号，从1开始
}

//TransferInfoArg 流转信息参数
type TransferInfoArg struct {
	BillID			string	`json:"ti_bill_id"`	//票据编号
	OldOwner		string	`json:"old_owner"`	//流转前票据所有者系统账号
	OldOwnerName	string	`json:"old_owner_name"`	//流转前票据所有者名称
	NewOwner		string	`json:"new_owner"`	//流转后票据所有者系统账号
	NewOwnerName	string	`json:"new_owner_name"`	//流转后票据所有者名称
}

//BillChild 拆分后父子票据关系基本结构
type BillChild struct {
	ParentID	string		`json:"cd_parent_id"`	//父票据号
	Childs		[]string	`json:"child_bills"`	//子票据号集合
}

//BillSplitInfo 票据拆分参数结构
type BillSplitInfoArg struct {
	BillID		string		`json:"bill_id"`	//票据号
	OwnerName	string		`json:"owner_name"`	//持票人名称
	Childs		[]BillChildArg	`json:"child_bills"`	//待拆分的票据
}

//BillChildArg 子票据参数结构
type BillChildArg struct {
	BillID		string	`json:"bill_id"`	//票据号
	Owner		string	`json:"owner"`		//持票人账号
	OwnerName	string	`json:"owner_name"`	//持票人名称
	Amount		float64	`json:"amount"`		//票据金额
}

//TableDataArg 表数据记录新增、修改及查询参数结构
type TableDataArg struct {
	TableName	string		`json:"table_name"`	//表名
	Data		interface{}	`json:"data"`	//数据
}

const (
	AnyError 	= 0
	ObjectExist = 1
	ObjectNew	= 2
)

//Tables above
var SF_TABLES = map[string]string{
	"bill": "BILL_",
	"loan": "LOAN_",
	"contract": "CNTR_",
	"bill_child": "BLCD_",
	"bill_transfer": "BLTF_",
	"loan_repayment": "LNRP_",
	"transferred_bill": "TFBL_",
}

type DocTable struct{
	Name string
	stub shim.ChaincodeStubInterface
}

func (dt DocTable) composeKey(id string) string {
	return SF_TABLES[dt.Name] + id
}

func (dt DocTable) GetObjectBytes(id string) ([]byte, error) {
	obj_bytes, err := dt.stub.GetState(dt.composeKey(id))

	if err != nil {
		return nil, err
	}

	return obj_bytes, nil

}

// pObj: pointer to individual object
func (dt DocTable) GetObject(id string, pObj interface{}) error {
	obj_bytes, err := dt.GetObjectBytes(id)
	
	if err != nil {
		return err
	} else if obj_bytes != nil {
		err = json.Unmarshal(obj_bytes, pObj)
		if err != nil {
			return err
		}
	}
	
	return nil
}

func (dt DocTable) SaveObject(id string, obj interface{}) error {
	obj_bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	err = dt.stub.PutState(dt.composeKey(id), obj_bytes)
	if err != nil {
		return err
	}
	
	return nil
}


func (dt DocTable) IsObjectExist(id string) (bool, error) {
	obj_bytes, err := dt.GetObjectBytes(id)
	
	if err != nil {
		return false, err
	} else if obj_bytes == nil {
		return false, nil
	}
	
	return true, nil
}

func (dt DocTable) IsTableExist() bool {
	_, exist := SF_TABLES[dt.Name]
	return exist
}

// chaincode response结构
type chaincodeRet struct {
	Code int    // 0 success otherwise 1
	Description  string //description
}

//SupplyFinance chaincode基本结构
type SupplyFinance struct {
}

func (bsi BillSplitInfoArg) SumAmountOfChildBill() float64{
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
	r.Description = des

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
	r.Description = des

	b, err := json.Marshal(r)

	if err != nil {
		fmt.Println("marshal Ret failed")
		return ""
	}
	return string(b[:])
}

// pObj: pointer to individual object
func NewObjectFromJsonString(jsonStr string, pObj interface{}) error {
	err := json.Unmarshal([]byte(jsonStr), pObj)

	if err != nil {
		return err
	}
	
	return nil
}

//Init chaincode基本接口
func (sfb *SupplyFinance) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

//Invoke chaincode基本接口
func (sfb *SupplyFinance) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()

	if function == "issueBill" {
		// 由合同关联或拆分票据而产生票据
		return sfb.issueBill(stub, args)
	} else if function == "endorseBill" {
		// 核心企业同意担保票据
		return sfb.endorseBill(stub, args)
	} else if function == "rejectBill" {
		// 核心企业拒绝担保票据
		return sfb.rejectBill(stub, args)
	} else if function == "issueContract" {
		// 上传并生成合同
		return sfb.issueContract(stub, args)
	} else if function == "endorseContract" {
		// 核心企业同意担保合同
		return sfb.endorseContract(stub, args)
	} else if function == "rejectContract" {
		// 核心企业拒绝担保合同
		return sfb.rejectContract(stub, args)
	} else if function == "transferBill" {
		// 流转票据
		return sfb.transferBill(stub, args)
	} else if function == "redeemBill" {
		// 已还款，赎回票据
		return sfb.redeemBill(stub, args)
	} else if function == "abolishBill" {
		// 票据持有人废弃票据
		return sfb.abolishBill(stub, args)
	} else if function == "splitBill" {
		// 票据持有人拆分票据
		return sfb.splitBill(stub, args)
	} else if function == "applyGuarantee" {
		// 申请贷款前，需要信用企业先担保贷款
		return sfb.applyGuarantee(stub, args)
	} else if function == "endorseLoan" {
		// 核心企业同意为供应商贷款担保 
		return sfb.endorseLoan(stub, args)
	} else if function == "rejectLoan" {
		// 核心企业拒绝为供应商贷款担保 
		return sfb.rejectLoan(stub, args)
	} else if function == "applyLoanAfterGuarantee" {
		// 担保成功后，票据持有人继续申请贷款
		return sfb.applyLoanAfterGuarantee(stub, args)
	} else if function == "applyLoan" {
		// 票据持有人申请贷款
		return sfb.applyLoan(stub, args)
	} else if function == "refuseLoan" {
		// 金融机构拒绝给票据持有人贷款
		return sfb.refuseLoan(stub, args)
	} else if function == "approveLoan" {
		// 金融机构同意给票据持有人贷款
		return sfb.approveLoan(stub, args)
	} else if function == "makeLoan" {
		// 金融机构同意贷款后放贷
		return sfb.makeLoan(stub, args)
	} else if function == "prepayLoan" {
		// 贷款人申请提前还款
		return sfb.prepayLoan(stub, args)
	} else if function == "repayLoan" {
		// 贷款人还款
		return sfb.repayLoan(stub, args)
	} else if function == "queryBillChilds" {
		// 查询票据拆分后的子票据ID集合
		return sfb.queryBillChilds(stub, args)
	} else if function == "queryByID" {
		// 按唯一ID查询单条记录
		return sfb.queryByID(stub, args)
	} else if function == "queryAll" {
		// 按条件查询多条记录
		return sfb.queryAll(stub, args)
	} else if function == "queryBillsWithPagination" {
		// 按条件分页查询
		return sfb.queryBillsWithPagination(stub, args)
	} else if function == "queryTXChainForBill" {
		// 查询票据或贷款的交易历史
		return sfb.queryTXChainForBill(stub, args)
	}

	res := getRetString(1, "Chaincode Unkown method!")

	return shim.Error(res)
}

//issueContract 上传并生成合同信息
// args: 0 - {Contract Object}
func (sfb *SupplyFinance) issueContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke issue contract args != 1")
		return shim.Error(res)
	}

	var ct Contract
	err := NewObjectFromJsonString(args[0], &ct)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke issueContract failed: %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}

	msg, ok := sfb.issueContractObj(stub, &ct, ContractUploaded)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

func (sfb *SupplyFinance) issueContractObj(stub shim.ChaincodeStubInterface, ct *Contract, init_state string) (string, bool){
	// ID唯一
	dt := DocTable{"contract", stub}
	
	exist, err := dt.IsObjectExist(ct.ContractID)
	if err != nil {
		return err.Error(), false
	} else if exist {
		res := fmt.Sprintf("Chaincode Invoke issueContractObj failed : the bill has existting, bill NO: %s", ct.ContractID)
		return res, false
	}

	// 设置状态
	ct.State = init_state

	// 保存
	err = dt.SaveObject(ct.ContractID, *ct)
	if err != nil {
		return err.Error(), false
	}

	return "invoke issue success", true
}

func (td TableDataArg) isTableExist() bool {
	_, exist := SF_TABLES[td.TableName]	
	return exist
}

//issueBill 票据发布
// args: 0 - {TableDataArg Object}
func (sfb *SupplyFinance) issueBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke issueBill args != 1")
		return shim.Error(res)
	}

/*	var td TableDataArg
	err := json.Unmarshal([]byte(args[0]), &td)
	if err != nil {
		res := getRetString(1, "Chaincode Invoke issueBill unmarshal failed")
		return shim.Error(res)
	}
	
	if !td.isTableExist() {
		res := fmt.Sprintf("Chaincode Invoke issueBill failed, due to table[%s] not exist", td.TableName)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var bill Bill
	// 注意：json的key名称包含下划线时，下面的转换会得不到预期的值
	// 用驼峰格式命名可解决
	if err := ms.Decode(td.Data, &bill); err != nil {
		res := getRetString(1, "Chaincode Invoke issueBill failed: unkown bill record format")
		return shim.Error(res)
	}
*/

	var bill Bill
	err := NewObjectFromJsonString(args[0], &bill)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke issueBill failed: %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}

	msg, ok := sfb.issueBillObj(stub, &bill, -1, Endorsed)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

func (sfb *SupplyFinance) issueBillObj(stub shim.ChaincodeStubInterface, bill *Bill, parent_split_count int32, init_state string) (string, bool){
	// 根据票号 查找是否票号已存在
	dt := DocTable{"bill", stub}
	exist, err := dt.IsObjectExist(bill.BillID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke issueBillObj failed : %s", err.Error())
		return res, false
	} else if exist {
		res := fmt.Sprintf("Chaincode Invoke issueBillObj failed : the bill has existting, bill NO: %s", bill.BillID)
		return res, false
	}

	// 设置票据的状态
	bill.State = init_state

	// 更新票据拆分次数
	bill.SplitCount = parent_split_count + 1

	// 保存票据
	err = dt.SaveObject(bill.BillID, *bill)
	if err != nil {
		return err.Error(), false
	}

	return "invoke issue success", true
}

//applyLoan 申请贷款
// args: 0 - {Loan Object}
func (sfb *SupplyFinance) applyLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	return tryPutApplyLoanObj(stub, args, LoanApplied)
}

//applyGuarantee 申请贷款，但需要信用企业先担保贷款
// args: 0 - {Loan Object}
func (sfb *SupplyFinance) applyGuarantee(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	return tryPutApplyLoanObj(stub, args, LoanGurantee)
}


func tryPutApplyLoanObj(stub shim.ChaincodeStubInterface, args []string, init_state string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke applyLoan args != 1")
		return shim.Error(res)
	}

	var ln Loan
	err := NewObjectFromJsonString(args[0], &ln)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke tryPutApplyLoanObj failed: %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}

	msg, ok := issueLoanObj(stub, &ln, init_state)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	msg, ok = tryUpdateBillForLoan(stub, ln.BillID,  Endorsed, BillLoanReady)
	if !ok {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

func issueLoanObj(stub shim.ChaincodeStubInterface, ln *Loan, init_state string) (string, bool){
	// 根据贷款编号 查找是否贷款申请已存在
	dt := DocTable{"loan", stub}
	exist, err := dt.IsObjectExist(ln.LoanID)
	if err != nil {
		return err.Error(), false
	} else if exist {
		res := fmt.Sprintf("Chaincode Invoke issueLoanObj failed : the bill has existting, bill NO: %s", ln.LoanID)
		return res, false
	}

	// 设置状态
	ln.State = init_state

	// 保存
	err = dt.SaveObject(ln.LoanID, *ln)
	if err != nil {
		return err.Error(), false
	}

	return "invoke success", true
}

func setLoanRepaymentThenPut(stub shim.ChaincodeStubInterface, lr *LoanRepayment, lra *LoanRepaymentArg) (string, bool){
	if lra != nil {
		lr.ActualRepaymentDate = lra.ActualRepaymentDate
		lr.ActualAmount		   = lra.ActualAmount		
		lr.AmountUnit	       = lra.AmountUnit	
		lr.ActualBankRate      = lra.ActualBankRate
		lr.ActualBankInterest  = lra.ActualBankInterest	
	}

	dt := DocTable{"loan_repayment", stub}
	// 保存
	err := dt.SaveObject(lr.LoanID, *lr)
	if err != nil {
		return err.Error(), false
	}
	
	return "invoke success", true
}

//repayLoan 还贷款
// args: 0 - {LoanRepaymentArg Object}
func (sfb *SupplyFinance) repayLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke repayLoan args count expecting 1")
		return shim.Error(res)
	}

	var lra LoanRepaymentArg
	err := json.Unmarshal([]byte(args[0]), &lra)
	if err != nil {
		res := getRetString(1, "Chaincode Invoke repayLoan unmarshal failed")
		return shim.Error(res)
	}
	
	dt := DocTable{"loan", stub}
	
	exist, err := dt.IsObjectExist(lra.LoanID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke repayLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke repayLoan failed : the loan is not existing, loan NO: %s", lra.LoanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var loan Loan
	dt.GetObject(lra.LoanID, &loan)

	if loan.BankName != lra.BankName {
		res := getRetString(1, "Chaincode Invoke repayLoan failed: guarantor's name is not same with current's")
		return shim.Error(res)
	}

	dt = DocTable{"loan_repayment", stub}	
	lr := LoanRepayment{LoanID: loan.LoanID}
	
	err = dt.GetObject(loan.LoanID, &lr)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke repayLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	lra.AmountUnit = loan.AmountUnit
	msg, ok2 := setLoanRepaymentThenPut(stub, &lr, &lra)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}
	
	msg, ok2 = setLoanStateThenPut(stub, &loan, LoanLoaned, LoanRepaid)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	dt = DocTable{"bill", stub}
	
	exist, err = dt.IsObjectExist(loan.BillID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke repayLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke repayLoan failed : the bill is not existing, bill NO: %s", loan.BillID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var bill Bill
	dt.GetObject(loan.BillID, &bill)

	msg, ok2 = setBillStateThenPut(stub, &bill, BillMorgaged, BillRedeemed)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//endorseLoan 担保贷款
// args: 0 - Loan ID; 1 -Guarantor Name
func (sfb *SupplyFinance) endorseLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke endorseLoan args count expecting 2")
		return shim.Error(res)
	}

	loanID := args[0]
	dt := DocTable{"loan", stub}
	
	exist, err := dt.IsObjectExist(loanID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke endorseLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke endorseLoan failed : the loan is not existing, loan NO: %s", loanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var loan Loan
	dt.GetObject(loanID, &loan)

	if ! loan.ValidateGuarantorName(args[1]) {
		res := getRetString(1, "Chaincode Invoke endorseLoan failed: guarantor's name is not same with current's")
		return shim.Error(res)
	}

	msg, ok2 := setLoanStateThenPut(stub, &loan, LoanGurantee, LoanApplied)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//rejectLoan 担保人拒绝担保贷款
// args: 0 - Loan ID; 1 -Guarantor Name; 2 - Refuse Reason
func (sfb *SupplyFinance) rejectLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		res := getRetString(1, "Chaincode Invoke rejectLoan args count expecting 3")
		return shim.Error(res)
	}

	loanID := args[0]
	dt := DocTable{"loan", stub}
	
	exist, err := dt.IsObjectExist(loanID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke rejectLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke rejectLoan failed : the loan is not existing, loan NO: %s", loanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var loan Loan
	dt.GetObject(loanID, &loan)

	if ! loan.ValidateGuarantorName(args[1]) {
		res := getRetString(1, "Chaincode Invoke rejectLoan failed: guarantor's name is not same with current's")
		return shim.Error(res)
	}

	loan.RefuseReason = args[2]
	msg, ok2 := setLoanStateThenPut(stub, &loan, LoanGurantee, Rejected)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	// TODO: 需不需修改票据状态为endorsed?

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//refuseLoan 金融机构拒绝贷款 
// args: 0 - {LoanResultArg object}
func (sfb *SupplyFinance) refuseLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke refuseLoan args != 1")
		return shim.Error(res)
	}

	var lr LoanResultArg
	err := json.Unmarshal([]byte(args[0]), &lr)

	if err != nil {
		res := getRetString(1, "Chaincode Invoke refuseLoan unmarshal failed")
		return shim.Error(res)
	}

	loanID := lr.LoanID
	dt := DocTable{"loan", stub}
	
	exist, err := dt.IsObjectExist(loanID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke refuseLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke refuseLoan failed : the loan is not existing, loan NO: %s", loanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var loan Loan
	dt.GetObject(loanID, &loan)

	if loan.OwnerName != lr.OwnerName {
		res := getRetString(1, "Chaincode Invoke refuseLoan failed: owner's name is not same with current's")
		return shim.Error(res)
	}

	loan.Bank = lr.Bank
	loan.BankName = lr.BankName
	loan.RefuseReason = lr.RefuseReason

	msg, ok2 := setLoanStateThenPut(stub, &loan, LoanApplied, LoanRefused)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	msg, ok2 = tryUpdateBillForLoan(stub, loan.BillID, BillLoanReady, Endorsed)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//makeLoan 金融机构同意贷款后放贷
// args: 0 - Loan ID; 1 -Bank Name; 2 - MakeLoan Date
func (sfb *SupplyFinance) makeLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		res := getRetString(1, "Chaincode Invoke makeLoan args count expecting 3")
		return shim.Error(res)
	}

	loanID := args[0]
	dt := DocTable{"loan", stub}
	
	exist, err := dt.IsObjectExist(loanID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke makeLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke makeLoan failed : the loan is not existing, loan NO: %s", loanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var loan Loan
	dt.GetObject(loanID, &loan)

	if ! loan.ValidateBankName(args[1]) {
		res := getRetString(1, "Chaincode Invoke makeLoan failed: bank's name is not same with current's")
		return shim.Error(res)
	}

	dt = DocTable{"loan_repayment", stub}	
	lr := LoanRepayment{LoanID: loan.LoanID}
	
	err = dt.GetObject(loan.LoanID, &lr)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke makeLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	makeLoanDate, err := strconv.ParseInt(args[2], 10, 64)
		if err != nil {
		return shim.Error("Chaincode Invoke makeLoan failed, due to string convert to int64")
	}
	lr.makeLoanDate = makeLoanDate
	
	msg, ok2 := setLoanRepaymentThenPut(stub, &lr, nil)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}
	
	msg, ok2 = setLoanStateThenPut(stub, &loan, LoanApproved, LoanLoaned)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//approveLoan 金融机构同意贷款
// args: 0 - {LoanResultArg object}
func (sfb *SupplyFinance) approveLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke approveLoan args != 1")
		return shim.Error(res)
	}

	var lr LoanResultArg
	err := json.Unmarshal([]byte(args[0]), &lr)

	if err != nil {
		res := getRetString(1, "Chaincode Invoke approveLoan unmarshal failed")
		return shim.Error(res)
	}

	loanID := lr.LoanID
	dt := DocTable{"loan", stub}
	
	exist, err := dt.IsObjectExist(loanID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke approveLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke approveLoan failed : the loan is not existing, loan NO: %s", loanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var loan Loan
	dt.GetObject(loanID, &loan)

	if ! loan.ValidateOwnerName(lr.OwnerName) {
		res := getRetString(1, "Chaincode Invoke approveLoan failed: owner's name is not same with current's")
		return shim.Error(res)
	}

	loan.Bank = lr.Bank
	loan.BankName = lr.BankName

	msg, ok2 := setLoanStateThenPut(stub, &loan, LoanApplied, LoanApproved)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	msg, ok2 = tryUpdateBillForLoan(stub, loan.BillID, BillLoanReady, BillMorgaged)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}
	
	lrp := LoanRepayment{LoanID: loan.LoanID, IsPrepayment: false}
	
	msg, ok2 = setLoanRepaymentThenPut(stub, &lrp, nil)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//prepayLoan 提前还款
// args: 0 - Loan ID; 1 - Owner Name;
func (sfb *SupplyFinance) prepayLoan(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke prepayLoan args != 2")
		return shim.Error(res)
	}

	loanID := args[0]
	dt := DocTable{"loan", stub}
	
	exist, err := dt.IsObjectExist(loanID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke prepayLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke prepayLoan failed : the loan is not existing, loan NO: %s", loanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var loan Loan
	dt.GetObject(loanID, &loan)
	
	if ! loan.ValidateOwnerName(args[1]) {
		res := getRetString(1, "Chaincode Invoke prepayLoan failed: loan owner's name is not same with current's")
		return shim.Error(res)
	}
	
	if ! loan.ValidateState(LoanApproved) && ! loan.ValidateState(LoanLoaned) {
		res := getRetString(1, "Chaincode Invoke prepayLoan failed: loan's state is not expected status, needed: 'approved' or 'loaned'")
		return shim.Error(res)
	}
	
	dt = DocTable{"loan_repayment", stub}	
	lr := LoanRepayment{LoanID: loan.LoanID}
	
	err = dt.GetObject(loan.LoanID, &lr)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke prepayLoan failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	if lr.IsPrepayment {
		res := fmt.Sprintf("Chaincode Invoke prepayLoan failed: The loan has been applied prepayment, loan NO: %s", loan.LoanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	lr.IsPrepayment = true
	msg, ok2 := setLoanRepaymentThenPut(stub, &lr, nil)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}
	
	res := getRetByte(0, msg)
	return shim.Success(res)
}

//applyLoanAfterGuarantee 贷款担保成功后，继续申请贷款
// args: 0 - Loan ID ; 1 - Owner Name
func (sfb *SupplyFinance) applyLoanAfterGuarantee(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke applyLoanAfterGuarantee args count expecting 2")
		return shim.Error(res)
	}

	loanID := args[0]
	dt := DocTable{"loan", stub}
	
	exist, err := dt.IsObjectExist(loanID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke applyLoanAfterGuarantee failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke applyLoanAfterGuarantee failed : the loan is not existing, loan NO: %s", loanID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var loan Loan
	dt.GetObject(loanID, &loan)
	
	if ! loan.ValidateOwnerName(args[1]) {
		res := getRetString(1, "Chaincode Invoke applyLoanAfterGuarantee failed: owner's name is not same with current's")
		return shim.Error(res)
	}

	msg, ok2 := setLoanStateThenPut(stub, &loan, Endorsed, LoanApplied)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}
	return shim.Success(nil)
}


func setLoanStateThenPut(stub shim.ChaincodeStubInterface, loan *Loan, expected_state, set_state string) (string, bool){
	// 检查票据当前状态
	if loan.State != expected_state {
		res := fmt.Sprintf("Chaincode Invoke failed: due to loan's state, current state: %s", loan.State)
		return res, false
	}

	// 更改票据状态
	loan.State = set_state

	dt := DocTable{"loan", stub}
	// 保存
	err := dt.SaveObject(loan.LoanID, *loan)
	if err != nil {
		return err.Error(), false
	}

	return "invoke success", true
}

//endorseContract 担保合同
//  args: 0 - Contract_No ; 1 - Drawee Name ; 2 - Bill ID
func (sfb *SupplyFinance) endorseContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		res := getRetString(1, "Chaincode Invoke endorse args count expecting 3")
		return shim.Error(res)
	}
	
	contractID := args[0]
	dt := DocTable{"contract", stub}
	
	exist, err := dt.IsObjectExist(contractID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke endorseContract failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke endorseContract failed : the contract is not existing, contract NO: %s", contractID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var ct Contract
	dt.GetObject(contractID, &ct)

	if ! ct.ValidateDraweeName(args[1]) {
		res := getRetString(1, "Chaincode Invoke endorseContract failed: Endorser is not same with current drawee")
		return shim.Error(res)
	}

	msg, ok2 := setContractStateThenPut(stub, &ct, ContractUploaded, Endorsed)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)

	}

	var bill Bill
	bill.ParentID = ct.ContractID
	bill.BillID = args[2]
	bill.Amount = ct.Amount
	bill.AmountUnit = ct.AmountUnit
	bill.IssueDate = ct.IssueDate
	bill.DueDate = ct.DueDate
	bill.PyeeName = ct.PyeeName
	bill.PyeeID = ct.PyeeID
	bill.PyeeAcct = ct.PyeeAcct
	bill.Drawee = ct.Drawee
	bill.DraweeName = ct.DraweeName
	bill.Issuer = ct.Issuer
	bill.IssuerName = ct.IssuerName
	bill.Owner = ct.Owner
	bill.OwnerName = ct.OwnerName

	msg, ok2 = sfb.issueBillObj(stub, &bill, -1, Endorsed)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

func setContractStateThenPut(stub shim.ChaincodeStubInterface, ct *Contract, expected_state, set_state string) (string, bool){
	// 检查票据当前状态
	if ct.State != expected_state {
		res := fmt.Sprintf("Chaincode Invoke setContractStateThenPut failed: due to contract's state, current state: %s", ct.State)
		return res, false
	}

	// 更改票据状态
	ct.State = set_state

	dt := DocTable{"contract", stub}
	// 保存
	err := dt.SaveObject(ct.ContractID, *ct)
	if err != nil {
		return err.Error(), false
	}
	
	return "invoke success", true
}

//transferBill 票据流转
//  args: 0 - {Transfer Info Object}
func (sfb *SupplyFinance) transferBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke transferBill args count expecting 1")
		return shim.Error(res)
	}
	
	var ti TransferInfoArg
	err := json.Unmarshal([]byte(args[0]), &ti)
	if err != nil {
		res := getRetString(1, "Chaincode Invoke transferBill unmarshal failed")
		return shim.Error(res)
	}
	
	// 根据票号取得票据
	dt := DocTable{"bill", stub}
	
	exist, err := dt.IsObjectExist(ti.BillID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke transferBill failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke transferBill failed : the bill is not existing, bill NO: %s", ti.BillID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var bill Bill
	dt.GetObject(ti.BillID, &bill)

	if ! bill.ValidateOwnerName(ti.OldOwnerName) {
		res := getRetString(1, "Chaincode Invoke transferBill failed: the owner of bill is not same with current's")
		return shim.Error(res)
	}
	
	if bill.ValidateOwnerName(ti.NewOwnerName) || bill.ValidateOwner(ti.NewOwner) {
		res := getRetString(1, "Chaincode Invoke transferBill failed: the bill should not transfer to self")
		return shim.Error(res)
	}
	
	if ! bill.ValidateState(Endorsed) {
		res := getRetString(1, "Chaincode Invoke transferBill failed: The bill can not be transferred, due to it's state.")
		return shim.Error(res)
	}

	// 根据票号取得票据流转信息
	dt = DocTable{"bill_transfer", stub}	
	bt := BillTransfer{BillID: ti.BillID, Count: 0, Transfers: make(map[int]TransferInfoArg)}
	
	err = dt.GetObject(ti.BillID, &bt)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke transferBill failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	// 保存票据流转信息
	ti.OldOwner = bill.Owner
	msg, ok2 := setBillTransferThenPut(stub, &bt, ti)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}
	
	msg, ok2 = setTransferredBillThenPut(stub, &bill)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}
	
		// 更新票据所有者
	bill.Owner = ti.NewOwner
	bill.OwnerName = ti.NewOwnerName
	bill.Transferred = true
	msg, ok2 = setBillStateThenPut(stub, &bill, Endorsed, Endorsed)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}
	
	res := getRetByte(0, "Invoke transferBill success")
	return shim.Success(res)
}

func setBillTransferThenPut(stub shim.ChaincodeStubInterface, bt *BillTransfer, ti TransferInfoArg) (string, bool){
	bt.Count += 1
	bt.Transfers[bt.Count] = ti

	dt := DocTable{"bill_transfer", stub}
	// 保存
	err := dt.SaveObject(bt.BillID, *bt)
	if err != nil {
		return err.Error(), false
	}

	return "invoke success", true
}

func setTransferredBillThenPut(stub shim.ChaincodeStubInterface, bill *Bill) (string, bool){
	dt := DocTable{"transferred_bill", stub}	
	tb := TransferredBill{Owner: bill.Owner}
	
	err := dt.GetObject(bill.Owner, &tb)
	if err != nil {
		return err.Error(), false
	}
	
	tb.Bills = append(tb.Bills, bill.BillID)

	dt = DocTable{"transferred_bill", stub}
	// 保存
	err = dt.SaveObject(tb.Owner, tb)
	if err != nil {
		return err.Error(), false
	}

	return "invoke success", true
}

//redeemBill 赎回票据
//  args: 0 - Bill_No ; 1 - Drawee Name 
func (sfb *SupplyFinance) redeemBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	return shim.Success(nil)
}

//endorseBill 担保票据
//  args: 0 - Bill_No ; 1 - Drawee Name 
func (sfb *SupplyFinance) endorseBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke endorseBill args count expecting 2")
		return shim.Error(res)
	}
	// 根据票号取得票据
	billID := args[0]
	dt := DocTable{"bill", stub}
	
	exist, err := dt.IsObjectExist(billID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke endorseBill failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke endorseBill failed : the bill is not existing, bill NO: %s", billID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var bill Bill
	dt.GetObject(billID, &bill)

	if ! bill.ValidateDraweeName(args[1]) {
		res := getRetString(1, "Chaincode Invoke endorseBill failed: Endorser is not same with current drawee")
		return shim.Error(res)
	}

	msg, ok2 := setBillStateThenPut(stub, &bill, BillIssued, Endorsed)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

func setBillStateThenPut(stub shim.ChaincodeStubInterface, bill *Bill, expected_state, set_state string) (string, bool){
	// 检查票据当前状态
	if bill.State != expected_state {
		res := fmt.Sprintf("Chaincode Invoke setBillStateThenPut failed: due to bill's state, current state: %s", bill.State)
		return res, false
	}

	// 更改票据状态
	bill.State = set_state

	dt := DocTable{"bill", stub}
	// 保存
	err := dt.SaveObject(bill.BillID, *bill)
	if err != nil {
		return err.Error(), false
	}

	return "invoke success", true
}

//rejectContract 拒绝担保合同
//  args: 0 - Contract_No ; 1 - Drawee Name ; 2 - Rejected Reason
func (sfb *SupplyFinance) rejectContract(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 3 {
		res := getRetString(1, "Chaincode Invoke rejectContract args count expecting 3")
		return shim.Error(res)
	}

	contractID := args[0]
	dt := DocTable{"contract", stub}
	
	exist, err := dt.IsObjectExist(contractID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke rejectContract failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke rejectContract failed : the contract is not existing, contract NO: %s", contractID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var contract Contract
	dt.GetObject(contractID, &contract)

	if ! contract.ValidateDraweeName(args[1]) {
		res := getRetString(1, "Chaincode Invoke rejectContract failed: Endorser is not same with current drawee")
		return shim.Error(res)
	}

	contract.RefuseReason = args[2]

	msg, ok2 := setContractStateThenPut(stub, &contract, ContractUploaded, Rejected)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

//rejectBill 拒绝担保票据
//  args: 0 - Bill_No ; 1 - Drawee Name 
func (sfb *SupplyFinance) rejectBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke rejectBill args count expecting 2")
		return shim.Error(res)
	}

	// 根据票号取得票据
	billID := args[0]
	dt := DocTable{"bill", stub}
	
	exist, err := dt.IsObjectExist(billID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke rejectBill failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke rejectBill failed : the bill is not existing, bill NO: %s", billID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var bill Bill
	dt.GetObject(billID, &bill)

	if ! bill.ValidateDraweeName(args[1]) {
		res := getRetString(1, "Chaincode Invoke rejectBill failed: Endorser is not same with current drawee")
		return shim.Error(res)
	}

	msg, ok2 := setBillStateThenPut(stub, &bill, BillIssued, Rejected)
	if !ok2 {
		res := getRetString(1, msg)
		return shim.Error(res)
	}

	res := getRetByte(0, msg)
	return shim.Success(res)
}

func tryUpdateBillForLoan(stub shim.ChaincodeStubInterface, bill_id string, expected_state, set_state string) (string, bool) {
	// 根据票号取得票据
	dt := DocTable{"bill", stub}
	
	exist, err := dt.IsObjectExist(bill_id)
	if err != nil {
		return err.Error(), false
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke tryUpdateBillForLoan failed : the bill is not existing, bill NO: %s", bill_id)
		return res, false
	}
	
	var bill Bill
	dt.GetObject(bill_id, &bill)

	if ! bill.ValidateDueDate() {
		return "Chaincode tryUpdateBillForLoan failed: the bill is expired", false
	}

	return setBillStateThenPut(stub, &bill, expected_state, set_state)
}

//abolishBill 作废票据
//  args: 0 - Bill_No ; 1 - Owner
func (sfb *SupplyFinance) abolishBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode Invoke abolishBill args count expecting 2")
		return shim.Error(res)
	}

	// 根据票号取得票据
	billID := args[0]
	dt := DocTable{"bill", stub}
	
	exist, err := dt.IsObjectExist(billID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke abolishBill failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke abolishBill failed : the bill is not existing, bill NO: %s", billID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	var bill Bill
	dt.GetObject(billID, &bill)

	if ! bill.ValidateOwnerName(args[1]) {
		res := getRetString(1, "Chaincode Invoke abolishBill failed: owner is not same with current owner")
		return shim.Error(res)
	}

	// 已经抵押贷款和被拆分的票据不运行作废
	if bill.ValidateState(BillMorgaged) || bill.ValidateState(BillSplit) {
		res := fmt.Sprintf("Chaincode Invoke abolishBill failed: The bill can not be abolished due to bill's state, current state: %s", bill.State)
		res = getRetString(1, res)
		return shim.Error(res)
	}

	// 更改票据状态为拒绝背书担保
	bill.State = BillAbolished

	// 保存
	err = dt.SaveObject(billID, bill)
	if err != nil {
		return shim.Error(err.Error())
	}

	res := getRetByte(0, "invoke accept success")

	return shim.Success(res)
}

/*splitBill 拆分票据
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
func (sfb *SupplyFinance) splitBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode Invoke splitBill args count expecting 1")
		return shim.Error(res)
	}

	var bsi BillSplitInfoArg
	err := json.Unmarshal([]byte(args[0]), &bsi)

	if err != nil {
		res := getRetString(1, "Chaincode Invoke splitBill unmarshal failed")
		return shim.Error(res)
	}

	var b Bill
	dt := DocTable{"bill", stub}
	
	exist, err := dt.IsObjectExist(bsi.BillID)
	if err != nil {
		res := fmt.Sprintf("Chaincode Invoke splitBill failed : %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	} else if !exist {
		res := fmt.Sprintf("Chaincode Invoke splitBill failed : the bill is not existing, bill NO: %s", bsi.BillID)
		res = getRetString(1, res)
		return shim.Error(res)
	}
	
	dt.GetObject(bsi.BillID, &b)

	if ! b.ValidateOwnerName(bsi.OwnerName) {
		res := getRetString(1, "Chaincode Invoke splitBill failed: owner is not same with current owner")
		return shim.Error(res)
	}

	if ! b.ValidateSplitCount(SplitThreshold) {
		res := fmt.Sprintf("Chaincode Invoke splitBill failed : the original bill has been spit up to max times, current threshold: %d", SplitThreshold)
		res = getRetString(1, res)
		return shim.Error(res)
	}

	// 只有通过背书担保的票据才能拆分
	if ! b.ValidateState(Endorsed) {
		res := fmt.Sprintf("Chaincode Invoke splitBill failed: The bill can not be split due to bill's state, current state: %s", b.State)
		res = getRetString(1, res)
		return shim.Error(res)
	}

	sumAmount := bsi.SumAmountOfChildBill()
	if b.Amount != sumAmount {
		res := fmt.Sprintf("Chaincode Invoke splitBill failed: The total amount of all child bills is not equal the parent's amount")
		res = getRetString(1, res)
		return shim.Error(res)
	}

	if len(bsi.Childs) < 2 {
		res := getRetString(1, "Chaincode Invoke splitBill failed: at least 2 Sub-Bills are required")
		return shim.Error(res)
	}

	child_bills := make([]string, 0)

	b_child := b
	b_child.ParentID = bsi.BillID
	for _, bc := range bsi.Childs {
		b_child.BillID= bc.BillID
		b_child.Owner = bc.Owner
		b_child.OwnerName = bc.OwnerName
		b_child.Amount = bc.Amount

		msg, ok := sfb.issueBillObj(stub, &b_child, b.SplitCount, Endorsed)
		if !ok {
			res := getRetString(1, msg)
			return shim.Error(res)
		}

		child_bills = append(child_bills, bc.BillID)
	}

	b.State = BillSplit
	
	dt = DocTable{"bill", stub}
	// 保存
	err = dt.SaveObject(b.BillID, b)
	if err != nil {
		return shim.Error(err.Error())
	}

	msg, ok := putBillChild(stub, bsi.BillID, child_bills)
	if !ok {
		return shim.Error(msg)
	}

	res := getRetByte(0, "invoke endorse success")
	return shim.Success(res)

}

func putBillChild(stub shim.ChaincodeStubInterface, parent_id string, child_bills []string) (string, bool) {
	var bc BillChild
	bc.ParentID = parent_id
	bc.Childs = child_bills

	dt := DocTable{"bill_child", stub}
	// 保存
	err := dt.SaveObject(bc.ParentID, bc)
	if err != nil {
		return err.Error(), false
	}

	return "Invoke success", true
}

//queryMarblesWithPagination 分页查询票据发起人、持有人、还款人的所有票据
//  0 - Issuer|Drawee|Owner ; 1 - count of page ; 2 - pagination bookmark
func (t *SupplyFinance) queryBillsWithPagination(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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

//queryAll 返回所有符合条件的记录
//  0 - {CouchDB selector query}
func (t *SupplyFinance) queryAll(stub shim.ChaincodeStubInterface, args []string) pb.Response {

	if len(args) < 1 {
		return shim.Error("Chaincode query[queryAll] failed: argument expecting 1")
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
		if i != 0 && b.Len() > 0 {
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

//queryTXChainForBill 根据Key查询交易链
//  0 - Table Name; 1 - ID ;
func (t *SupplyFinance) queryTXChainForBill(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Chaincode query[queryTXChainForBill] failed: argument expecting 2")
	}

	table_name := args[0]
	id := args[1]
	key := SF_TABLES[table_name] + id

	resultsIterator, err := stub.GetHistoryForKey(key)
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

// 根据ID查询拆分后的子票据
// args: 0 - Table Name; 1 - id
func (sfb *SupplyFinance) queryBillChilds(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		res := getRetString(1, "Chaincode queryBillChilds args != 1")
		return shim.Error(res)
	}

	table_name := args[0]
	id := args[1]
	key := SF_TABLES[table_name] + id
	
	objBytes, err := stub.GetState(key)
	if  err != nil {
		res := fmt.Sprintf("Chaincode queryBillChilds failed: %s", err.Error())
		res = getRetString(1, res)
		return shim.Error(res)
	}

	if objBytes == nil {
		return shim.Success([]byte("{}"))
	}

	return shim.Success(objBytes)
}

// 根据ID查询记录
// args: 0 - Table Name; 1 - id
func (sfb *SupplyFinance) queryByID(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		res := getRetString(1, "Chaincode queryByID args != 2")
		return shim.Error(res)
	}
	
	tableName := args[0]
	id := args[1]
	dt := DocTable{tableName, stub}
	
	if exist := dt.IsTableExist(); !exist {
		res := fmt.Sprintf("Chaincode queryByID failed: the table[%s] is not exist.", tableName)
		res = getRetString(1, res)
		return shim.Error(res)
	}

	objBytes, err := dt.GetObjectBytes(id)
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

func main() {
	if err := shim.Start(new(SupplyFinance)); err != nil {
		fmt.Printf("Error starting SupplyFinance: %s", err)
	}
}
