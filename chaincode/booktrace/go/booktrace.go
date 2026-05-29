package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
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

func getPaperEmissionFactor(paperType string) float64 {
	switch paperType {
	case "再生纸":
		return 0.6
	case "环保纸":
		return 0.8
	case "铜版纸":
		return 1.8
	default:
		return 1.0
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

func (s *SmartContract) BatchCreateBooks(ctx contractapi.TransactionContextInterface, batchId string, booksJSON string, paperType string, printEnergy string, bookWeight string) error {
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

	printEnergyFloat, _ := strconv.ParseFloat(printEnergy, 64)
	bookWeightFloat, _ := strconv.ParseFloat(bookWeight, 64)

	for _, bookInput := range input.Books {
		err := s.CreateBook(ctx, bookInput.ISBN, bookInput.BookName, bookInput.Author,
			bookInput.Publisher, bookInput.PublishDate, bookInput.Category,
			fmt.Sprintf("%.2f", bookInput.Price), fmt.Sprintf("%d", bookInput.Quantity),
			batchId, paperType, printEnergy, bookWeight)
		if err != nil {
			return fmt.Errorf("创建图书 %s 失败: %v", bookInput.ISBN, err)
		}
	}
	// 修正：使用正确解析的浮点数
	_ = printEnergyFloat
	_ = bookWeightFloat
	return nil
}

func (s *SmartContract) CreateBook(ctx contractapi.TransactionContextInterface, isbn string, bookName string,
	author string, publisher string, publishDate string, category string, price string, quantity string,
	batchId string, paperType string, printEnergy string, bookWeight string) error {
	exists, err := s.BookExists(ctx, isbn)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("图书 %s 已存在", isbn)
	}

	priceFloat, _ := strconv.ParseFloat(price, 64)
	quantityInt, _ := strconv.Atoi(quantity)
	printEnergyFloat, _ := strconv.ParseFloat(printEnergy, 64)
	bookWeightFloat, _ := strconv.ParseFloat(bookWeight, 64)

	now := time.Now()
	antiCode := generateAntiCounterfeitCode(isbn)

	paperFactor := getPaperEmissionFactor(paperType)
	printEmission := paperFactor * printEnergyFloat/100 * 0.5 * bookWeightFloat

	book := Book{
		ISBN:           isbn,
		BookName:       bookName,
		Author:         author,
		Publisher:      publisher,
		PublishDate:    publishDate,
		Category:       category,
		Price:          priceFloat,
		Quantity:       quantityInt,
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
			PrintEnergy:        printEnergyFloat,
			TransportDistance:  0,
			TransportMode:      "",
			BookWeight:         bookWeightFloat,
		},
		Comments:           []Comment{},
		SecondHandListings: []SecondHandListing{},
		FlowHistory: []Flow{
			{
				State:     "图书信息已录入",
				OrgName:   "出版社",
				OptTime:   now.Format("2006-01-02 15:04:05"),
				Remark:    fmt.Sprintf("出版社 %s 录入图书，数量: %d", publisher, quantityInt),
				BatchId:   batchId,
				Timestamp: now.Unix(),
				Emissions: printEmission,
			},
		},
	}

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

// UpdateBookState 更新图书状态（支持增加数量）
func (s *SmartContract) UpdateBookState(ctx contractapi.TransactionContextInterface, isbn string, newState string, orgName string, remark string, batchId string, transportDistance string, transportMode string, quantity string) error {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}
	if !book.IsActive {
		return fmt.Errorf("图书 %s 已被删除", isbn)
	}

	// ===== 关键修复：允许同状态更新（用于增加上架数量） =====
	addQuantity, _ := strconv.Atoi(quantity)
	
	// 如果是同一状态转换（如"书店上架" -> "书店上架"），只增加数量
	if book.CurrentState == newState {
		if addQuantity > 0 {
			book.Quantity += addQuantity
			now := time.Now()
			book.FlowHistory = append(book.FlowHistory, Flow{
				State:     newState,
				OrgName:   orgName,
				OptTime:   now.Format("2006-01-02 15:04:05"),
				Remark:    fmt.Sprintf("%s 增加上架数量 %d 本，当前库存 %d", orgName, addQuantity, book.Quantity),
				BatchId:   batchId,
				Timestamp: now.Unix(),
			})
			book.UpdatedAt = now.Format("2006-01-02 15:04:05")
			bookBytes, _ := json.Marshal(book)
			return ctx.GetStub().PutState(isbn, bookBytes)
		}
		return fmt.Errorf("同状态更新需要指定 quantity 参数")
	}

	// 不同状态转换的验证
	validTransitions := map[string][]string{
		"图书信息已录入": {"印刷完成"},
		"印刷完成":     {"出厂分发"},
		"出厂分发":     {"批发商入库"},
		"批发商入库":    {"配送到门店"},
		"配送到门店":    {"书店上架"},
		"书店上架":     {"已售出", "书店上架"}, // 允许书店上架 -> 书店上架（补货）
		"已售出":       {},                    // 已售出不能转换
		"二手转售":     {"已售出"},              // 二手转售可以转为已售出
	}

	if allowedNext, ok := validTransitions[book.CurrentState]; ok {
		valid := false
		for _, state := range allowedNext {
			if state == newState {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("无效状态流转: %s -> %s，允许的状态: %v", book.CurrentState, newState, allowedNext)
		}
	}

	// 运输碳排放计算
	var emission float64
	var distance float64
	fmt.Sscanf(transportDistance, "%f", &distance)
	if newState == "批发商入库" && distance > 0 {
		modeFactors := map[string]float64{"公路": 1.0, "铁路": 0.5, "航空": 3.0, "海运": 0.3}
		factor := modeFactors[transportMode]
		if factor == 0 {
			factor = 1.0
		}
		emission = distance * factor * book.CarbonFootprint.BookWeight * 0.05
		book.CarbonFootprint.TransportDistance = distance
		book.CarbonFootprint.TransportMode = transportMode
		book.CarbonFootprint.TransportEmissions = emission
		book.CarbonFootprint.TotalEmissions = book.CarbonFootprint.PrintEmissions + emission
	}

	// 如果是配送到门店，更新数量
	if newState == "书店上架" && addQuantity > 0 {
		book.Quantity += addQuantity
	}
	// 如果是配送到门店但没有指定数量，使用当前所有数量
	if newState == "书店上架" && addQuantity == 0 && book.Quantity > 0 {
		addQuantity = book.Quantity
	}

	newFlow := Flow{
		State:     newState,
		OrgName:   orgName,
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    fmt.Sprintf("%s - %s", remark, func() string {
			if addQuantity > 0 {
				return fmt.Sprintf("，数量: %d", addQuantity)
			}
			return ""
		}()),
		BatchId:   batchId,
		Timestamp: time.Now().Unix(),
		Emissions: emission,
	}
	book.FlowHistory = append(book.FlowHistory, newFlow)
	book.CurrentState = newState
	book.LastOrg = orgName
	book.LastTime = newFlow.OptTime
	book.UpdatedAt = newFlow.OptTime

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

func (s *SmartContract) BuyBook(ctx contractapi.TransactionContextInterface, isbn string, userId string, userName string, count string) error {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}

	if book.CurrentState != "书店上架" {
		return fmt.Errorf("该书当前不可购买")
	}

	buyCount, _ := strconv.Atoi(count)
	if book.Quantity < buyCount {
		return fmt.Errorf("库存不足，当前库存: %d，请求购买: %d", book.Quantity, buyCount)
	}

	book.Quantity -= buyCount
	book.TotalSold += buyCount

	if book.Quantity == 0 {
		book.CurrentState = "已售出"
	}

	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	book.FlowHistory = append(book.FlowHistory, Flow{
		State:     "已售出",
		OrgName:   "普通用户",
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    fmt.Sprintf("用户 %s 购买 %s 本", userName, count),
		BatchId:   "buy_" + time.Now().Format("20060102150405"),
		Timestamp: time.Now().Unix(),
	})

	bookBytes, _ := json.Marshal(book)
	return ctx.GetStub().PutState(isbn, bookBytes)
}

func (s *SmartContract) AddReview(ctx contractapi.TransactionContextInterface, isbn string, userId string, userName string, rating string, reviewType string, content string, hasBought string) error {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return err
	}

	if hasBought != "true" {
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

	ratingInt, _ := strconv.Atoi(rating)
	commentId := generateDeterministicId(userId, isbn)

	comment := Comment{
		Id:        commentId,
		UserId:    userId,
		UserName:  userName,
		Rating:    ratingInt,
		Type:      reviewType,
		Content:   content,
		Time:      time.Now().Format("2006-01-02 15:04:05"),
		HasBought: hasBought == "true",
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

// VerifyAntiCounterfeit 防伪码验证
func (s *SmartContract) VerifyAntiCounterfeit(ctx contractapi.TransactionContextInterface, isbn string, ip string, location string, userAgent string) (map[string]interface{}, error) {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return nil, err
	}

	scanRecord := ScanRecord{
		ScanTime:  time.Now().Format("2006-01-02 15:04:05"),
		IP:        ip,
		Location:  location,
		UserAgent: userAgent,
	}
	book.AntiCounterfeit.ScanRecords = append(book.AntiCounterfeit.ScanRecords, scanRecord)
	book.AntiCounterfeit.ScanCount++

	if book.AntiCounterfeit.ScanCount >= 3 {
		book.AntiCounterfeit.IsAbnormal = true
	}

	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
	bookBytes, _ := json.Marshal(book)
	err = ctx.GetStub().PutState(isbn, bookBytes)
	if err != nil {
		return nil, err
	}

	var message string
	if book.AntiCounterfeit.IsAbnormal {
		message = "⚠️ 防伪码异常！该码已被多次查询，请警惕假冒产品！"
	} else if book.AntiCounterfeit.ScanCount == 1 {
		message = "✅ 验证成功！这是首次查询，商品为正品。"
	} else {
		message = fmt.Sprintf("✅ 验证成功！这是第 %d 次查询。", book.AntiCounterfeit.ScanCount)
	}

	result := map[string]interface{}{
		"isValid":    true,
		"bookName":   book.BookName,
		"scanCount":  book.AntiCounterfeit.ScanCount,
		"isAbnormal": book.AntiCounterfeit.IsAbnormal,
		"message":    message,
	}
	return result, nil
}

// ===== 关键修复：ListSecondHandBook 不再修改图书状态 =====
func (s *SmartContract) ListSecondHandBook(ctx contractapi.TransactionContextInterface, isbn string, sellerId string, sellerName string, price string, condition string) (string, error) {
	book, err := s.QueryBookByISBN(ctx, isbn)
	if err != nil {
		return "", err
	}

	// 检查是否重复上架
	for _, listing := range book.SecondHandListings {
		if listing.SellerId == sellerId && listing.Status == "available" {
			return "", fmt.Errorf("您已经上架了这本书，请勿重复上架")
		}
	}

	hash := sha256.Sum256([]byte(isbn + sellerId + time.Now().Format("20060102150405")))
	listingId := hex.EncodeToString(hash[:])[:20]
	priceFloat, _ := strconv.ParseFloat(price, 64)

	listing := SecondHandListing{
		Id:            listingId,
		ISBN:          isbn,
		BookName:      book.BookName,
		SellerId:      sellerId,
		SellerName:    sellerName,
		Price:         priceFloat,
		Condition:     condition,
		OriginalPrice: book.Price,
		Status:        "available",
		CreateTime:    time.Now().Format("2006-01-02 15:04:05"),
		UpdateTime:    time.Now().Format("2006-01-02 15:04:05"),
	}
	book.SecondHandListings = append(book.SecondHandListings, listing)
	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	// ===== 修复：不再修改 book.CurrentState =====
	// 二手书转售不影响书店的链上图书状态
	// 只记录流转历史，不改变状态

	book.FlowHistory = append(book.FlowHistory, Flow{
		State:     "二手转售",
		OrgName:   "普通用户",
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    fmt.Sprintf("用户 %s 上架二手书，价格 ¥%s（不影响书店库存）", sellerName, price),
		BatchId:   "secondhand_" + time.Now().Format("20060102150405"),
		Timestamp: time.Now().Unix(),
	})

	bookBytes, _ := json.Marshal(book)
	err = ctx.GetStub().PutState(isbn, bookBytes)
	return listingId, err
}

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
	book.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	// ===== 修复：购买二手书也不修改图书主状态 =====
	// 只记录流转历史

	book.FlowHistory = append(book.FlowHistory, Flow{
		State:     "二手转售",
		OrgName:   "二手市场",
		OptTime:   time.Now().Format("2006-01-02 15:04:05"),
		Remark:    fmt.Sprintf("用户 %s 从 %s 购买二手书，价格 ¥%.2f（不影响书店库存）", buyerName, targetListing.SellerName, targetListing.Price),
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
