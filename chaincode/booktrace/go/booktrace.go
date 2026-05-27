package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

type AntiCounterfeit struct {
	Code        string       `json:"code"`
	ScanCount   int          `json:"scanCount"`
	ScanRecords []ScanRecord `json:"scanRecords"`
	IsAbnormal  bool         `json:"isAbnormal"`
	CreatedAt   string       `json:"createdAt"`
}

type ScanRecord struct {
	ScanTime  string `json:"scanTime"`
	IP        string `json:"ip"`
	Location  string `json:"location"`
	UserAgent string `json:"userAgent"`
}

type CarbonFootprint struct {
	PrintEmissions     float64 `json:"printEmissions"`
	TransportEmissions float64 `json:"transportEmissions"`
	TotalEmissions     float64 `json:"totalEmissions"`
	PaperType          string  `json:"paperType"`
	PrintEnergy        float64 `json:"printEnergy"`
	TransportDistance  float64 `json:"transportDistance"`
	TransportMode      string  `json:"transportMode"`
	BookWeight         float64 `json:"bookWeight"`
}

type Comment struct {
	Id        int64  `json:"id"`
	UserId    string `json:"userId"`
	UserName  string `json:"userName"`
	Rating    int    `json:"rating"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Time      string `json:"time"`
	HasBought bool   `json:"hasBought"`
}

type Flow struct {
	State     string  `json:"state"`
	OrgName   string  `json:"orgName"`
	Operator  string  `json:"operator"`
	OptTime   string  `json:"optTime"`
	Remark    string  `json:"remark"`
	BatchId   string  `json:"batchId"`
	Timestamp int64   `json:"timestamp"`
	Emissions float64 `json:"emissions"`
}

type SecondHandListing struct {
	Id            string  `json:"id"`
	ISBN          string  `json:"isbn"`
	BookName      string  `json:"bookName"`
	SellerId      string  `json:"sellerId"`
	SellerName    string  `json:"sellerName"`
	Price         float64 `json:"price"`
	Condition     string  `json:"condition"`
	OriginalPrice float64 `json:"originalPrice"`
	Status        string  `json:"status"`
	CreateTime    string  `json:"createTime"`
	UpdateTime    string  `json:"updateTime"`
}

type Book struct {
	ISBN               string              `json:"isbn"`
	BookName           string              `json:"bookName"`
	Author             string              `json:"author"`
	Publisher          string              `json:"publisher"`
	PublishDate        string              `json:"publishDate"`
	Category           string              `json:"category"`
	Price              float64             `json:"price"`
	Quantity           int                 `json:"quantity"`
	TotalSold          int                 `json:"totalSold"`
	BatchId            string              `json:"batchId"`
	CurrentState       string              `json:"currentState"`
	LastOrg            string              `json:"lastOrg"`
	LastTime           string              `json:"lastTime"`
	FlowHistory        []Flow              `json:"flowHistory"`
	AntiCounterfeit    AntiCounterfeit     `json:"antiCounterfeit"`
	CarbonFootprint    CarbonFootprint     `json:"carbonFootprint"`
	Comments           []Comment           `json:"comments"`
	GoodReviews        int                 `json:"goodReviews"`
	BadReviews         int                 `json:"badReviews"`
	ReviewCount        int                 `json:"reviewCount"`
	SecondHandListings []SecondHandListing `json:"secondHandListings"`
	CreatedAt          string              `json:"createdAt"`
	UpdatedAt          string              `json:"updatedAt"`
	IsActive           bool                `json:"isActive"`
}

type SmartContract struct {
	contractapi.Contract
}

func generateAntiCounterfeitCode(isbn string) string {
	hash := sha256.Sum256([]byte(isbn))
	return hex.EncodeToString(hash[:])[:16]
}

func getPaperEmission(paperType string) float64 {
	switch paperType {
	case "再生纸":
		return 0.8
	case "环保纸":
		return 1.0
	case "铜版纸":
		return 2.0
	default:
		return 1.5
	}
}

func generateDeterministicId(userId string, isbn string) int64 {
	hash := sha256.Sum256([]byte(userId + isbn))
	var id int64 = 0
	for i := 0; i < 8; i++ {
		id = (id << 8) | int64(hash[i])
	}
	if id < 0 {
		id = -id
	}
	return id
}

func (s *SmartContract) BatchCreateBooks(ctx contractapi.TransactionContextInterface, batchId string, booksJSON string, paperType string, printEnergy float64, bookWeight float64) error {
	var input struct {
		Books []struct {
			ISBN        string  `json:"isbn"`
			BookName    string  `json:"bookName"`
			Author      string  `json:"author"`
			Publisher   string  `json:"publisher"`
			PublishDate string  `json:"publishDate"`
			Category    string  `json:"category"`
			Price       float64 `json:"price"`
			Quantity    int     `json:"quantity"`
		} `json:"books"`
	}
	err := json.Unmarshal([]byte(booksJSON), &input)
	if err != nil {
		return fmt.Errorf("解析批量数据失败: %v", err)
	}

	for _, bookInput := range input.Books {
		err := s.CreateBook(ctx, bookInput.ISBN, bookInput.BookName, bookInput.Author,
			bookInput.Publisher, bookInput.PublishDate, bookInput.Category,
			bookInput.Price, bookInput.Quantity, batchId, paperType, printEnergy, bookWeight)
		if err != nil {
			return fmt.Errorf("创建图书 %s 失败: %v", bookInput.ISBN, err)
		}
	}
	return nil
}

func (s *SmartContract) CreateBook(ctx contractapi.TransactionContextInterface, isbn string, bookName string,
	author string, publisher string, publishDate string, category string, price float64, quantity int,
	batchId string, paperType string, printEnergy float64, bookWeight float64) error {
	exists, err := s.BookExists(ctx, isbn)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("图书 %s 已存在", isbn)
	}

	now := time.Now()
	antiCode := generateAntiCounterfeitCode(isbn)

	paperEmission := getPaperEmission(paperType)
	printEmission := paperEmission * printEnergy / 100

	book := Book{
		ISBN:           isbn,
		BookName:       bookName,
		Author:         author,
		Publisher:      publisher,
		PublishDate:    publishDate,
		Category:       category,
		Price:          price,
		Quantity:       quantity,
		TotalSold:      0,
		BatchId:        batchId,
		CurrentState:   "图书信息已录入",
		LastOrg:        "出版社",
		LastTime:       now.Format("2006-01-02 15:04:05"),
		CreatedAt:      now.Format("2006-01-02 15:04:05"),
		UpdatedAt:      now.Format("2006-01-02 15:04:05"),
		IsActive:       true,
		GoodReviews:    0,
		BadReviews:     0,
		ReviewCount:    0,
		AntiCounterfeit: AntiCounterfeit{
			Code:        antiCode,
			ScanCount:   0,
			ScanRecords: []ScanRecord{},
			IsAbnormal:  false,
			CreatedAt:   now.Format("2006-01-02 15:04:05"),
		},
		CarbonFootprint: CarbonFootprint{
			PrintEmissions:     printEmission,
			TransportEmissions: 0,
			TotalEmissions:     printEmission,
			PaperType:          paperType,
			PrintEnergy:        printEnergy,
			TransportDistance:  0,
			TransportMode:      "",
			BookWeight:         bookWeight,
		},
		Comments:           []Comment{},
		SecondHandListings: []SecondHandListing{},
		FlowHistory: []Flow{
			{
				State:     "图书信息已录入",
				OrgName:   "出版社",
				OptTime:   now.Format("2006-01-02 15:04:05"),
				Remark:    fmt.Sprintf("出版社 %s 录入图书，数量: %d", publisher, quantity),
				BatchId:   batchId,
				Timestamp: now.Unix(),
				Emissions: printEmission,
			},
		},
	}

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

func (s *SmartContract) UpdateBookState(ctx contractapi.TransactionContextInterface, isbn string, newState string, orgName string, remark string, batchId string, transportDistance float64, transportMode string) error {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}
	if !book.IsActive {
		return fmt.Errorf("图书 %s 已被删除", isbn)
	}

	validTransitions := map[string][]string{
		"图书信息已录入": {"印刷完成"},
		"印刷完成":     {"出厂分发"},
		"出厂分发":     {"批发商入库"},
		"批发商入库":    {"配送到门店"},
		"配送到门店":    {"书店上架"},
		"书店上架":     {"已售出"},
	}

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

	var emission float64
	if newState == "批发商入库" && transportDistance > 0 {
		modeFactors := map[string]float64{"公路": 1.0, "铁路": 0.5, "航空": 3.0, "海运": 0.3}
		factor := modeFactors[transportMode]
		if factor == 0 {
			factor = 1.0
		}
		emission = transportDistance * 0.05 * factor * book.CarbonFootprint.BookWeight
		book.CarbonFootprint.TransportDistance = transportDistance
		book.CarbonFootprint.TransportMode = transportMode
		book.CarbonFootprint.TransportEmissions = emission
		book.CarbonFootprint.TotalEmissions = book.CarbonFootprint.PrintEmissions + emission
	}

	newFlow := Flow{
		State:     newState,
		OrgName:   orgName,
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    remark,
		BatchId:   batchId,
		Timestamp: time.Now().Unix(),
		Emissions: emission,
	}
	book.FlowHistory = append(book.FlowHistory, newFlow)
	book.CurrentState = newState
	book.LastOrg = orgName
	book.LastTime = newFlow.OptTime
	book.UpdatedAt = newFlow.OptTime

	// 如果是售出，减少库存（但保留状态，除非库存为0）
	if newState == "已售出" && book.Quantity > 0 {
		book.Quantity--
		book.TotalSold++
		if book.Quantity == 0 {
			book.CurrentState = "已售出"
		}
	}

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

// BuyBook 购买图书（只减少库存，不改变状态，除非库存为0）
func (s *SmartContract) BuyBook(ctx contractapi.TransactionContextInterface, isbn string, userId string, userName string, count int) error {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}

	if book.CurrentState != "书店上架" {
		return fmt.Errorf("该书当前不可购买")
	}

	if book.Quantity < count {
		return fmt.Errorf("库存不足，当前库存: %d，请求购买: %d", book.Quantity, count)
	}

	book.Quantity -= count
	book.TotalSold += count

	if book.Quantity == 0 {
		book.CurrentState = "已售出"
	}

	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	book.FlowHistory = append(book.FlowHistory, Flow{
		State:     "已售出",
		OrgName:   "普通用户",
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    fmt.Sprintf("用户 %s 购买 %d 本", userName, count),
		BatchId:   "buy_" + time.Now().Format("20060102150405"),
		Timestamp: time.Now().Unix(),
	})

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

func (s *SmartContract) AddReview(ctx contractapi.TransactionContextInterface, isbn string, userId string, userName string, rating int, reviewType string, content string, hasBought bool) error {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}

	if !hasBought {
		return fmt.Errorf("只有购买过该书的用户才能评价")
	}

	for _, comment := range book.Comments {
		if comment.UserId == userId {
			return fmt.Errorf("您已经评价过这本书了")
		}
	}

	if reviewType != "good" && reviewType != "bad" {
		return fmt.Errorf("评价类型只能是 good（好评）或 bad（差评）")
	}

	commentId := generateDeterministicId(userId, isbn)

	comment := Comment{
		Id:        commentId,
		UserId:    userId,
		UserName:  userName,
		Rating:    rating,
		Type:      reviewType,
		Content:   content,
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		HasBought: hasBought,
	}
	book.Comments = append(book.Comments, comment)

	if reviewType == "good" {
		book.GoodReviews++
	} else {
		book.BadReviews++
	}
	book.ReviewCount = len(book.Comments)
	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

func (s *SmartContract) GetReviewStats(ctx contractapi.TransactionContextInterface, isbn string) (map[string]interface{}, error) {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return nil, err
	}

	goodRate := float64(0)
	if book.ReviewCount > 0 {
		goodRate = float64(book.GoodReviews) / float64(book.ReviewCount) * 100
	}

	stats := map[string]interface{}{
		"goodReviews":  book.GoodReviews,
		"badReviews":   book.BadReviews,
		"totalReviews": book.ReviewCount,
		"goodRate":     goodRate,
	}
	return stats, nil
}

// ListSecondHandBook 上架二手书（使用确定性ID，用户购买后即可转售）
func (s *SmartContract) ListSecondHandBook(ctx contractapi.TransactionContextInterface, isbn string, sellerId string, sellerName string, price float64, condition string) (string, error) {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return "", err
	}

	// 检查卖家是否购买过这本书（只看购买记录，不管库存状态）
	hasBought := false
	for _, flow := range book.FlowHistory {
		if flow.State == "已售出" && flow.OrgName == "普通用户" {
			hasBought = true
			break
		}
	}

	if !hasBought {
		return "", fmt.Errorf("您没有购买过这本书，无法转售")
	}

	// 检查是否已经在售
	for _, listing := range book.SecondHandListings {
		if listing.SellerId == sellerId && listing.Status == "available" {
			return "", fmt.Errorf("您已经上架了这本书，请勿重复上架")
		}
	}

	// 使用确定性ID
	hash := sha256.Sum256([]byte(isbn + sellerId))
	listingId := hex.EncodeToString(hash[:])[:20]

	listing := SecondHandListing{
		Id:            listingId,
		ISBN:          isbn,
		BookName:      book.BookName,
		SellerId:      sellerId,
		SellerName:    sellerName,
		Price:         price,
		Condition:     condition,
		OriginalPrice: book.Price,
		Status:        "available",
		CreateTime:    time.Now().Format("2006-01-02 15:04:05"),
		UpdateTime:    time.Now().Format("2006-01-02 15:04:05"),
	}
	book.SecondHandListings = append(book.SecondHandListings, listing)
	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	if book.CurrentState != "二手转售" {
		book.CurrentState = "二手转售"
	}

	book.FlowHistory = append(book.FlowHistory, Flow{
		State:     "二手转售",
		OrgName:   "普通用户",
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    fmt.Sprintf("用户 %s 上架二手书，价格 ¥%.2f", sellerName, price),
		BatchId:   "secondhand_" + time.Now().Format("20060102150405"),
		Timestamp: time.Now().Unix(),
	})

	bookBytes, _ := json.Marshal(book)
	err = ctx.GetStub().PutState(isbn, bookBytes)
	return listingId, err
}

// BuySecondHandBook 购买二手书
func (s *SmartContract) BuySecondHandBook(ctx contractapi.TransactionContextInterface, isbn string, listingId string, buyerId string, buyerName string) error {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}

	var targetListing *SecondHandListing
	for i := range book.SecondHandListings {
		if book.SecondHandListings[i].Id == listingId && book.SecondHandListings[i].Status == "available" {
			targetListing = &book.SecondHandListings[i]
			break
		}
	}
	if targetListing == nil {
		return fmt.Errorf("该二手书不存在或已售出")
	}

	targetListing.Status = "sold"
	targetListing.UpdateTime = time.Now().Format("2006-01-02 15:04:05")
	book.CurrentState = "已售出"
	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	book.FlowHistory = append(book.FlowHistory, Flow{
		State:     "二手转售",
		OrgName:   "二手市场",
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    fmt.Sprintf("用户 %s 从 %s 购买二手书，价格 ¥%.2f", buyerName, targetListing.SellerName, targetListing.Price),
		Timestamp: time.Now().Unix(),
	})

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

func (s *SmartContract) GetSecondHandListings(ctx contractapi.TransactionContextInterface) ([]SecondHandListing, error) {
	books, err := s.QueryAllBooks(ctx)
	if err != nil {
		return nil, err
	}

	var listings []SecondHandListing
	for _, book := range books {
		for _, listing := range book.SecondHandListings {
			if listing.Status == "available" {
				listings = append(listings, listing)
			}
		}
	}
	return listings, nil
}

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

func (s *SmartContract) GetStatistics(ctx contractapi.TransactionContextInterface) (map[string]interface{}, error) {
	books, err := s.QueryAllBooks(ctx)
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	stateCount := make(map[string]int)
	var totalEmissions float64
	for _, book := range books {
		stateCount[book.CurrentState]++
		totalEmissions += book.CarbonFootprint.TotalEmissions
	}
	stats["totalBooks"] = len(books)
	stats["stateCount"] = stateCount
	stats["totalCarbonEmissions"] = totalEmissions
	return stats, nil
}

func (s *SmartContract) BookExists(ctx contractapi.TransactionContextInterface, isbn string) (bool, error) {
	bookBytes, err := ctx.GetStub().GetState(isbn)
	if err != nil {
		return false, err
	}
	return bookBytes != nil, nil
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
