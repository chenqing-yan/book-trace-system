package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ==================== 数据结构定义 ====================

// Flow 流转记录
type Flow struct {
	State     string `json:"state"`
	OrgName   string `json:"orgName"`
	Operator  string `json:"operator"`  // 操作人
	OptTime   string `json:"optTime"`
	Remark    string `json:"remark"`
	Timestamp int64  `json:"timestamp"` // Unix时间戳，用于时间范围查询
}

// Book 图书结构（增强版）
type Book struct {
	ISBN         string `json:"isbn"`
	BookName     string `json:"bookName"`
	Author       string `json:"author"`
	Publisher    string `json:"publisher"`
	PublishDate  string `json:"publishDate"`
	Category     string `json:"category"`     // 分类：科技/文学/教育等
	Price        string `json:"price"`        // 定价
	CurrentState string `json:"currentState"`
	LastOrg      string `json:"lastOrg"`
	LastTime     string `json:"lastTime"`
	FlowHistory  []Flow `json:"flowHistory"`
	CreatedAt    string `json:"createdAt"`    // 创建时间
	UpdatedAt    string `json:"updatedAt"`    // 更新时间
	IsActive     bool   `json:"isActive"`     // 是否有效（软删除标记）
}

// BatchCreateInput 批量创建输入
type BatchCreateInput struct {
	Books []BookInput `json:"books"`
}

// BookInput 创建图书输入
type BookInput struct {
	ISBN        string `json:"isbn"`
	BookName    string `json:"bookName"`
	Author      string `json:"author"`
	Publisher   string `json:"publisher"`
	PublishDate string `json:"publishDate"`
	Category    string `json:"category"`
	Price       string `json:"price"`
}

// Statistics 统计数据
type Statistics struct {
	TotalBooks     int            `json:"totalBooks"`
	StateCount     map[string]int `json:"stateCount"`
	PublisherCount map[string]int `json:"publisherCount"`
}

// ==================== 智能合约定义 ====================

type SmartContract struct {
	contractapi.Contract
}

// ==================== 图书创建功能 ====================

// CreateBook 出版社录入新书
func (s *SmartContract) CreateBook(ctx contractapi.TransactionContextInterface, isbn string, bookName string, author string, publisher string, publishDate string, category string, price string) error {
	// 权限检查：只有出版社可以创建
	orgName, err := s.getOrgName(ctx)
	if err != nil {
		return err
	}
	if orgName != "Org1MSP" && orgName != "publisher" {
		return fmt.Errorf("权限不足：只有出版社可以创建图书，当前组织: %s", orgName)
	}

	// 检查是否已存在
	exists, err := s.BookExists(ctx, isbn)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("图书 %s 已存在", isbn)
	}

	// 获取操作人身份
	operator := s.getOperator(ctx)

	now := time.Now()
	book := Book{
		ISBN:         isbn,
		BookName:     bookName,
		Author:       author,
		Publisher:    publisher,
		PublishDate:  publishDate,
		Category:     category,
		Price:        price,
		CurrentState: "图书信息已录入",
		LastOrg:      orgName,
		LastTime:     now.Format("2006-01-02 15:04:05"),
		CreatedAt:    now.Format("2006-01-02 15:04:05"),
		UpdatedAt:    now.Format("2006-01-02 15:04:05"),
		IsActive:     true,
		FlowHistory: []Flow{
			{
				State:     "图书信息已录入",
				OrgName:   orgName,
				Operator:  operator,
				OptTime:   now.Format("2006-01-02 15:04:05"),
				Remark:    fmt.Sprintf("出版社 %s 录入图书", publisher),
				Timestamp: now.Unix(),
			},
		},
	}

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

// BatchCreateBooks 批量创建图书
func (s *SmartContract) BatchCreateBooks(ctx contractapi.TransactionContextInterface, booksJSON string) error {
	var input BatchCreateInput
	err := json.Unmarshal([]byte(booksJSON), &input)
	if err != nil {
		return fmt.Errorf("解析批量数据失败: %v", err)
	}

	for _, book := range input.Books {
		err = s.CreateBook(ctx, book.ISBN, book.BookName, book.Author, book.Publisher, book.PublishDate, book.Category, book.Price)
		if err != nil {
			return fmt.Errorf("创建图书 %s 失败: %v", book.ISBN, err)
		}
	}
	return nil
}

// ==================== 图书更新功能 ====================

// UpdateBookInfo 更新图书基础信息（出版社专用）
func (s *SmartContract) UpdateBookInfo(ctx contractapi.TransactionContextInterface, isbn string, bookName string, author string, category string, price string) error {
	orgName, err := s.getOrgName(ctx)
	if err != nil {
		return err
	}
	if orgName != "Org1MSP" && orgName != "publisher" {
		return fmt.Errorf("权限不足：只有出版社可以修改图书信息")
	}

	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}
	if !book.IsActive {
		return fmt.Errorf("图书 %s 已被删除", isbn)
	}

	if bookName != "" {
		book.BookName = bookName
	}
	if author != "" {
		book.Author = author
	}
	if category != "" {
		book.Category = category
	}
	if price != "" {
		book.Price = price
	}
	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	// 追加更新记录
	book.FlowHistory = append(book.FlowHistory, Flow{
		State:   "图书信息已更新",
		OrgName: orgName,
		OptTime: time.Now().Format("2006-01-02 15:04:05"),
		Remark:  fmt.Sprintf("更新图书信息：书名=%s, 作者=%s, 分类=%s, 价格=%s", bookName, author, category, price),
	})

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

// UpdateBookState 更新图书流转状态
func (s *SmartContract) UpdateBookState(ctx contractapi.TransactionContextInterface, isbn string, newState string, remark string) error {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}
	if !book.IsActive {
		return fmt.Errorf("图书 %s 已被删除", isbn)
	}

	orgName, err := s.getOrgName(ctx)
	if err != nil {
		return err
	}
	operator := s.getOperator(ctx)

	// 状态流转映射（增强版，包含更多状态）
	validTransitions := map[string][]string{
		"图书信息已录入":   {"印刷中", "印刷完成", "已下架"},
		"印刷中":        {"印刷完成"},
		"印刷完成":       {"出厂分发", "印刷中"},
		"出厂分发":       {"批发商入库", "印刷完成"},
		"批发商入库":      {"配送到门店", "已下架"},
		"配送到门店":      {"书店上架", "批发商入库"},
		"书店上架":       {"已售出", "已下架", "二手转售"},
		"已售出":        {"二手转售"},
		"二手转售":       {"书店上架", "已下架"},
		"已下架":        {"已销毁"},
		"已销毁":        {},
	}

	// 检查状态流转是否合法
	if allowedNext, ok := validTransitions[book.CurrentState]; ok {
		valid := false
		for _, state := range allowedNext {
			if state == newState {
				valid = true
				break
			}
		}
		if !valid && len(allowedNext) > 0 {
			return fmt.Errorf("无效状态流转: %s -> %s，允许的状态: %v", book.CurrentState, newState, allowedNext)
		}
	}

	newFlow := Flow{
		State:     newState,
		OrgName:   orgName,
		Operator:  operator,
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    remark,
		Timestamp: time.Now().Unix(),
	}
	book.FlowHistory = append(book.FlowHistory, newFlow)
	book.CurrentState = newState
	book.LastOrg = orgName
	book.LastTime = newFlow.OptTime
	book.UpdatedAt = newFlow.OptTime

	// 如果是销毁或下架，标记为非活跃
	if newState == "已下架" || newState == "已销毁" {
		book.IsActive = false
	}

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

// DeleteBook 软删除图书（管理员/出版社权限）
func (s *SmartContract) DeleteBook(ctx contractapi.TransactionContextInterface, isbn string, reason string) error {
	orgName, err := s.getOrgName(ctx)
	if err != nil {
		return err
	}
	if orgName != "Org1MSP" && orgName != "publisher" && orgName != "Org2MSP" {
		return fmt.Errorf("权限不足：只有出版社或监管机构可以删除图书")
	}

	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}

	book.IsActive = false
	book.FlowHistory = append(book.FlowHistory, Flow{
		State:   "已删除",
		OrgName: orgName,
		OptTime: time.Now().Format("2006-01-02 15:04:05"),
		Remark:  fmt.Sprintf("删除原因: %s", reason),
	})
	book.CurrentState = "已删除"

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

// ==================== 查询功能 ====================

// QueryBookByISBN 根据ISBN查询完整溯源信息
func (s *SmartContract) QueryBookByISBN(ctx contractapi.TransactionContextInterface, isbn string) (*Book, error) {
	bookBytes, err := ctx.GetStub().GetState(isbn)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}
	if bookBytes == nil {
		return nil, fmt.Errorf("图书 %s 不存在", isbn)
	}
	var book Book
	err = json.Unmarshal(bookBytes, &book)
	return &book, err
}

// QueryAllBooks 查询所有活跃图书
func (s *SmartContract) QueryAllBooks(ctx contractapi.TransactionContextInterface) ([]*Book, error) {
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var books []*Book
	for resultsIterator.HasNext() {
		kvResult, err := resultsIterator.Next()
		if err != nil {
			continue
		}
		var book Book
		err = json.Unmarshal(kvResult.Value, &book)
		if err == nil && book.IsActive {
			books = append(books, &book)
		}
	}
	return books, nil
}

// QueryBooksByState 按状态查询图书
func (s *SmartContract) QueryBooksByState(ctx contractapi.TransactionContextInterface, state string) ([]*Book, error) {
	allBooks, err := s.QueryAllBooks(ctx)
	if err != nil {
		return nil, err
	}
	var result []*Book
	for _, book := range allBooks {
		if book.CurrentState == state {
			result = append(result, book)
		}
	}
	return result, nil
}

// QueryBooksByPublisher 按出版社查询图书
func (s *SmartContract) QueryBooksByPublisher(ctx contractapi.TransactionContextInterface, publisher string) ([]*Book, error) {
	allBooks, err := s.QueryAllBooks(ctx)
	if err != nil {
		return nil, err
	}
	var result []*Book
	for _, book := range allBooks {
		if book.Publisher == publisher {
			result = append(result, book)
		}
	}
	return result, nil
}

// QueryBooksByTimeRange 按时间范围查询（最近N天内流转的图书）
func (s *SmartContract) QueryBooksByTimeRange(ctx contractapi.TransactionContextInterface, days int) ([]*Book, error) {
	allBooks, err := s.QueryAllBooks(ctx)
	if err != nil {
		return nil, err
	}
	
	now := time.Now().Unix()
	threshold := now - int64(days*24*3600)
	
	var result []*Book
	for _, book := range allBooks {
		// 检查最后一个流程的时间
		if len(book.FlowHistory) > 0 {
			lastFlow := book.FlowHistory[len(book.FlowHistory)-1]
			if lastFlow.Timestamp >= threshold {
				result = append(result, book)
			}
		}
	}
	return result, nil
}

// QueryBookHistory 查询图书完整历史（别名，功能同QueryBookByISBN）
func (s *SmartContract) QueryBookHistory(ctx contractapi.TransactionContextInterface, isbn string) (*Book, error) {
	return s.QueryBookByISBN(ctx, isbn)
}

// ==================== 统计功能 ====================

// GetStatistics 获取统计数据
func (s *SmartContract) GetStatistics(ctx contractapi.TransactionContextInterface) (*Statistics, error) {
	allBooks, err := s.QueryAllBooks(ctx)
	if err != nil {
		return nil, err
	}
	
	stats := &Statistics{
		TotalBooks:     len(allBooks),
		StateCount:     make(map[string]int),
		PublisherCount: make(map[string]int),
	}
	
	for _, book := range allBooks {
		stats.StateCount[book.CurrentState]++
		stats.PublisherCount[book.Publisher]++
	}
	
	return stats, nil
}

// GetBookLifecycleDuration 计算图书从创建到当前状态的时长（天数）
func (s *SmartContract) GetBookLifecycleDuration(ctx contractapi.TransactionContextInterface, isbn string) (int64, error) {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return 0, err
	}
	
	// 解析创建时间
	createdAt, err := time.Parse("2006-01-02 15:04:05", book.CreatedAt)
	if err != nil {
		return 0, err
	}
	
	// 解析最后更新时间
	updatedAt, err := time.Parse("2006-01-02 15:04:05", book.UpdatedAt)
	if err != nil {
		return 0, err
	}
	
	duration := updatedAt.Sub(createdAt)
	return int64(duration.Hours() / 24), nil
}

// ==================== 辅助功能 ====================

// BookExists 判断图书是否存在
func (s *SmartContract) BookExists(ctx contractapi.TransactionContextInterface, isbn string) (bool, error) {
	bookBytes, err := ctx.GetStub().GetState(isbn)
	if err != nil {
		return false, err
	}
	return bookBytes != nil, nil
}

// getOrgName 获取调用者组织名称
func (s *SmartContract) getOrgName(ctx contractapi.TransactionContextInterface) (string, error) {
	clientIdentity := ctx.GetClientIdentity()
	mspid, err := clientIdentity.GetMSPID()
	if err != nil {
		return "", err
	}
	return mspid, nil
}

// getOperator 获取操作人（从证书中提取）
func (s *SmartContract) getOperator(ctx contractapi.TransactionContextInterface) string {
	clientIdentity := ctx.GetClientIdentity()
	id, err := clientIdentity.GetID()
	if err != nil {
		return "unknown"
	}
	// 简化显示，只取最后部分
	parts := strings.Split(id, "::")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return id
}

func main() {
	chaincode, err := contractapi.NewChaincode(&SmartContract{})
	if err != nil {
		fmt.Printf("创建链码失败: %s", err)
		return
	}
	if err := chaincode.Start(); err != nil {
		fmt.Printf("启动链码失败: %s", err)
	}
}
