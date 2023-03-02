package main

import (
	"bytes"
	"database/sql"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	generator "github.com/angelodlfrtr/go-invoice-generator"
	creditcard "github.com/durango/go-credit-card"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/melbahja/got"
)

//##############################################################################################
//Various Struct that are utilized throughout code from inputs, storing data, etc..
type OrderInfo struct {
	OrderReports   []OrderReports `json:"OrderReports"`
	CreditCardData CreditCardData `json:"CreditCardData"`
}

type OrderReports struct {
	ReportAddresses            []ReportAddresses `json:"ReportAddresses"`
	BuildingID                 string            `json:"BuildingId"`
	PrimaryProductID           int               `json:"PrimaryProductId"`
	DeliveryProductID          int               `json:"DeliveryProductId"`
	MeasurementInstructionType int               `json:"MeasurementInstructionType"`
	ClaimNumber                string            `json:"ClaimNumber"`
	ClaimInfo                  string            `json:"ClaimInfo"`
	BatchID                    string            `json:"BatchId"`
	CatID                      string            `json:"CatId"`
	ChangesInLast4Years        bool              `json:"ChangesInLast4Years"`
	PONumber                   string            `json:"PONumber"`
	Comments                   string            `json:"Comments"`
	ReferenceID                string            `json:"ReferenceId"`
	InsuredName                string            `json:"InsuredName"`
}

type ReportAddresses struct {
	Address                  string `json:"Address"`
	City                     string `json:"City"`
	State                    string `json:"State"`
	Zip                      string `json:"Zip"`
	AddressType              int    `json:"AddressType"`
	VerifierUsedID           int    `json:"VerifierUsedId"`
	MapperUsedID             int    `json:"MapperUsedId"`
	VerificationResultTypeID int    `json:"VerificationResultTypeId"`
}

type OrderStats struct {
	OrderID   int   `json:"OrderId"`
	ReportIds []int `json:"ReportIds"`
}

type input struct {
	reportNum string
}

type queryResult struct {
	street       string
	city         string
	state        string
	zipcode      string
	azimuth      string
	tilt         string
	solar_annual float64
	ac_annual    float64
	reportId     string
}

type PageVariables struct {
	Address    string
	Ac_annual  float64
	ReportId   string
	TopImage   string
	NorthImage string
	SouthImage string
	EastImage  string
	WestImage  string
	JsonMes    []ReportResult
}

type Response struct {
	Inputs struct {
		address        string `json:"address"`
		SystemCapacity string `json:"system_capacity"`
		Lat            string `json:"lat"`
		Lon            string `json:"lon"`
		Azimuth        string `json:"azimuth"`
		Tilt           string `json:"tilt"`
		ArrayType      string `json:"array_type"`
		ModuleType     string `json:"module_type"`
		Losses         string `json:"losses"`
	} `json:"inputs"`
	Errors   []interface{} `json:"errors"`
	Warnings []interface{} `json:"warnings"`
	Version  string        `json:"version"`
	SscInfo  struct {
		Version int    `json:"version"`
		Build   string `json:"build"`
	} `json:"ssc_info"`
	StationInfo struct {
		Lat               float64 `json:"lat"`
		Lon               float64 `json:"lon"`
		Elev              float64 `json:"elev"`
		Tz                int     `json:"tz"`
		Location          string  `json:"location"`
		City              string  `json:"city"`
		State             string  `json:"state"`
		SolarResourceFile string  `json:"solar_resource_file"`
		Distance          int     `json:"distance"`
	} `json:"station_info"`
	Outputs struct {
		AcMonthly      []float64 `json:"ac_monthly"`
		PoaMonthly     []float64 `json:"poa_monthly"`
		SolradMonthly  []float64 `json:"solrad_monthly"`
		DcMonthly      []float64 `json:"dc_monthly"`
		AcAnnual       float64   `json:"ac_annual"`
		SolradAnnual   float64   `json:"solrad_annual"`
		CapacityFactor float64   `json:"capacity_factor"`
	} `json:"outputs"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	AsClientID   string `json:"as:client_id"`
	Issued       string `json:".issued"`
	Expires      string `json:".expires"`
}

type Link struct {
	Links []Links `json:"Links"`
}
type Links struct {
	Link            string    `json:"Link"`
	ExpireTimestamp time.Time `json:"ExpireTimestamp"`
	FileType        string    `json:"FileType"`
}

type ReportResult struct {
	Designator    string `json:"designator"`
	Unroundedsize string `json:"unroundedsize"`
	Pitch         string `json:"pitch"`
	PitchDeg      string `json:"pitchDeg"`
	Orientation   string `json:"orientation"`
	Tsrf          string `json:"TSRF"`
	Sa            string `json:"SA"`
	SunHours      string `json:"sunhours"`
}

type ExportEV struct {
	Reportid string   `json:"reportid"`
	Location Location `json:"location"`
	Roofs    []Roofs  `json:"roofs"`
}
type Location struct {
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Northorientation float64 `json:"northorientation"`
	Address          string  `json:"address"`
	City             string  `json:"city"`
	Postal           string  `json:"postal"`
	State            string  `json:"state"`
}
type Irradiance struct {
	Tsrf float64 `json:"TSRF"`
	Sa   float64 `json:"SA"`
}
type Roofs struct {
	Designator    string       `json:"designator"`
	Unroundedsize string       `json:"unroundedsize"`
	Pitch         string       `json:"pitch"`
	PitchDeg      string       `json:"pitchDeg"`
	ID            string       `json:"id"`
	Orientation   float64      `json:"orientation"`
	Irradiance    []Irradiance `json:"irradiance"`
}

type PaymentInfo struct {
	ExpireMonth int
	ExpireYear  int
	CardNum     string
	CardType    int
}

type Address struct {
	Street    string
	City      string
	State     string
	Zip       string
	TypeRep   string
	Email     string
	FirstName string
	LastName  string
}

type CreditCardData struct {
	CardFirstName    string `json:"CardFirstName"`
	CardLastName     string `json:"CardLastName"`
	ExpirationMonth  int    `json:"ExpirationMonth"`
	ExpirationYear   int    `json:"ExpirationYear"`
	CreditCardNumber string `json:"CreditCardNumber"`
	CreditCardType   int    `json:"CreditCardType"`
}

//##############################################################################################

//Session Variable for redirecting
var (
	store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
)

//This Function checks if user report number exist after order was places and check if its ready
func lookUpPage(w http.ResponseWriter, r *http.Request) {
	var key = securecookie.GenerateRandomKey(32)

	os.Setenv(string(key), "SESSION_KEY")
	session, _ := store.Get(r, "cookie-name")

	session.Values["authenticated"] = true
	if r.Method == "GET" {
		t, _ := template.ParseFiles("html/formpage.html")
		t.Execute(w, nil)
	} else {
		r.ParseForm()
		var userInput input
		userInput.reportNum = r.FormValue("address")
		db, _ := connectDb()
		reportType := checkDb(db, userInput)
		token, _ := eagleViewToken()
		if reportType.Valid {
			var reportData []byte
			var reportLength int64
			var resp string

			if reportType.String == "Basic" {
				reportData, reportLength, resp = getReportFile(token, userInput.reportNum, 3, 2)
				if resp == "200 OK" && reportLength > 0 {
					downloadPDF(w, r, reportData)
				}
			} else {
				_, reportLength, resp = getReportFile(token, userInput.reportNum, 75, 2)
				if resp == "200 OK" && reportLength > 0 {
					session.Values["reportId"] = userInput.reportNum
					session.Save(r, w)
					http.Redirect(w, r, "/reportDisplay", http.StatusFound)
				}
			}
		} else {
			http.Redirect(w, r, "/formpage", http.StatusFound)
		}
		db.Close()
	}
}

//This Function place order for report and inputs data for NREL based on retrieved values
func order(token Token, addressInput Address, payment PaymentInfo, db *sql.DB) OrderStats {
	order, response := placeOrder(token, addressInput, payment)

	if response == "200 OK" {
		userDataToDb(addressInput, db, order)
		responseObject, _ := NRELData(addressInput)
		nrelToDb(responseObject, db, addressInput)
	}
	return order
}

//This function stores user information into the database
func userDataToDb(userInput Address, db *sql.DB, order OrderStats) {
	insert, err := db.Query("INSERT INTO OrderHistory VALUES (?,?,?,?,?,?,?,?,?)", userInput.FirstName, userInput.LastName, userInput.Email, userInput.Street, userInput.City, userInput.State, userInput.Zip, order.ReportIds[0], userInput.TypeRep)
	if err != nil {
		panic(err.Error())
	}
	insert.Close()
}

//This function establishes connection to the AWS DB instance
func connectDb() (*sql.DB, error) {
	//For establishing connection to created aws db
	//AWS Master User with Endpoint placed after the @ symbol and db instance name after / symbol
	db, err := sql.Open("mysql", "master:Facompilers@tcp(database-1.cu9z37ygfbm2.us-west-1.rds.amazonaws.com:3306)/database-1")

	if err != nil {
		fmt.Print(err.Error())
	}

	return db, err
}

//This function checks if the report type of the report from the report id
func checkDb(db *sql.DB, userInput input) sql.NullString {
	res, err := db.Query("SELECT reportType FROM OrderHistory where reportId=(?)", userInput.reportNum)
	if err != nil {
		panic(err.Error())
	}
	var reportId sql.NullString
	for res.Next() {
		err = res.Scan(&reportId)
		if err != nil {
			panic(err.Error())
		}
	}
	return reportId
}

//This function checks if the order exist when the user tries to places an order, so they do not reorder
func checkExistingOrder(db *sql.DB, address Address) sql.NullString {
	res, err := db.Query("SELECT reportId FROM OrderHistory where street=(?) and city=(?) and state=(?) and zipcode=(?)", address.Street, address.City, address.State, address.Zip)
	if err != nil {
		panic(err.Error())
	}
	var reportId sql.NullString
	for res.Next() {
		err = res.Scan(&reportId)
		if err != nil {
			panic(err.Error())
		}
	}
	return reportId
}

//This function retrieves data from NREL API
func NRELData(address Address) (Response, string) {
	result := strings.ReplaceAll(fmt.Sprintf("%s,%s,%s %s", address.Street, address.City, address.State, address.Zip), " ", "")
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("https://developer.nrel.gov/api/pvwatts/v6.json?api_key=bBU3vPPeFsRjGPWEjNUcU0Z4RNpvMZ4ufmlXiYCF&address=%s&system_capacity=0.08&azimuth=180&tilt=40&array_type=1&module_type=1&losses=10", result), nil)
	if err != nil {
		log.Print(err.Error())
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Print(err.Error())
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Print(err.Error())
	}

	var responseObject Response
	json.Unmarshal((bodyBytes), &responseObject)
	return responseObject, resp.Status
}

//This function insert data into the NREL table
func nrelToDb(responseObject Response, db *sql.DB, address Address) {
	insert, err := db.Query("INSERT INTO NREL VALUES (?,?,?,?,?,?,?,?,?,?,?)", address.Street, address.City, address.State, address.Zip, responseObject.StationInfo.Lat, responseObject.StationInfo.Lon, responseObject.Inputs.Azimuth, responseObject.Inputs.Tilt, responseObject.Outputs.SolradAnnual, responseObject.Outputs.AcAnnual)

	if err != nil {
		panic(err.Error())
	}

	defer insert.Close()
}

//This Function is used generate authentication token based on eagleview credentials
func eagleViewToken() (Token, string) {
	const clientSecret = "eZ6FNfStCJs3epr8tCCgXuMD8cgTeDb3WJxcWFt9jAbTzWLsmw6rY7pT8rsDRnHM"
	const SourceID = "36c184a9-1686-4761-976b-8e4ba4ed4f1b "
	const Endpoint = "https://webservices.eagleview.com/Token"
	sEnc := b64.StdEncoding.EncodeToString([]byte(SourceID + ":" + clientSecret))
	data := url.Values{}
	data.Add("grant_type", "password&username=ryonfaroughi@gmail.com&password=EagleView@1")

	req, err := http.NewRequest("POST", Endpoint, bytes.NewBuffer([]byte("grant_type=password&username=ryonfaroughi@gmail.com&password=EagleView@1")))
	if err != nil {
		log.Print(err.Error())
	}
	req.Header.Add("Host", "webservices.eagleview.com")
	req.Header.Add("Authorization", "Basic "+sEnc)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err.Error())
	}

	//fmt.Println(resp.Status)
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Print(err.Error())
	}

	var TokenObject Token
	json.Unmarshal((bodyBytes), &TokenObject)
	return TokenObject, resp.Status
}

//This function places order for user for select report
func placeOrder(TokenObject Token, address Address, payment PaymentInfo) (OrderStats, string) {
	var reportType int

	if address.TypeRep == "Basic" {
		reportType = 11
	} else if address.TypeRep == "Advanced" {
		reportType = 62
	}

	fmt.Printf("REPORT TYPE: %d\n\n", reportType)
	orderData := OrderInfo{
		OrderReports: []OrderReports{{
			ReportAddresses: []ReportAddresses{{
				Address:                  address.Street,
				City:                     address.City,
				State:                    address.State,
				Zip:                      address.Zip,
				AddressType:              1,
				VerifierUsedID:           1,
				MapperUsedID:             1,
				VerificationResultTypeID: 1}},
			BuildingID:                 "0",
			PrimaryProductID:           reportType,
			DeliveryProductID:          8,
			MeasurementInstructionType: 1,
			ClaimNumber:                "",
			ClaimInfo:                  "",
			BatchID:                    "",
			ChangesInLast4Years:        true,
			PONumber:                   "",
			Comments:                   "Roof Report",
			ReferenceID:                "",
			InsuredName:                ""}},
		CreditCardData: CreditCardData{
			CardFirstName:    address.FirstName,
			CardLastName:     address.LastName,
			ExpirationMonth:  payment.ExpireMonth,
			ExpirationYear:   payment.ExpireYear,
			CreditCardNumber: payment.CardNum,
			CreditCardType:   payment.CardType,
		}}

	jsonData, err := json.Marshal(orderData)
	if err != nil {
		log.Println(err.Error())
	}
	jsonByte := []byte(jsonData)
	const Endpoint = "https://webservices-integrations.eagleview.com/v2/Order/PlaceOrder"
	req, err := http.NewRequest("POST", Endpoint, bytes.NewBuffer([]byte(jsonByte)))
	if err != nil {
		log.Print(err.Error())
	}
	req.Header.Add("Host", "webservices-integrations.eagleview.com")
	req.Header.Add("Authorization", TokenObject.TokenType+" "+TokenObject.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err.Error())
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Print(err.Error())
	}

	var order OrderStats
	json.Unmarshal((bodyBytes), &order)

	return order, resp.Status
}

/**
Use EagleView API to get Image data and return it as 64encoded bytes
**/
func getReportImage(TokenObject Token, reportId string, imageType int) ([]byte, string) {
	var Endpoint = "https://webservices.eagleview.com/v1/File/GetReportFile?reportId=" + reportId + "&fileType=" + strconv.Itoa(imageType) + "&fileFormat=1" //+ strconv.Itoa(imageType)
	req, err := http.NewRequest("GET", Endpoint, nil)
	if err != nil {
		//Handle Error
		log.Print(err.Error())
	}
	req.Header.Add("Host", "webservices.eagleview.com")
	req.Header.Add("Authorization", TokenObject.TokenType+" "+TokenObject.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err.Error())
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Print("Error reading http response body\n")
	}

	//Convert bytes to images
	byteArray := []byte(bodyBytes)
	return byteArray, resp.Status
}

//This function Retrieves various reports files from eagleview
func getReportFile(TokenObject Token, reportId string, reportType int, fileType int) ([]byte, int64, string) {
	var Endpoint = "https://webservices.eagleview.com/v1/File/GetReportFile?reportId=" + reportId + "&fileType=" + strconv.Itoa(reportType) + "&fileFormat=" + strconv.Itoa(fileType)
	req, err := http.NewRequest("GET", Endpoint, nil)
	if err != nil {
		log.Print(err.Error())
	}
	req.Header.Add("Host", "webservices.eagleview.com")
	req.Header.Add("Authorization", TokenObject.TokenType+" "+TokenObject.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err.Error())
	}

	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Print(err.Error())
	}
	byteArray := []byte(bodyBytes)
	//fmt.Println(string(byteArray))
	return byteArray, resp.ContentLength, resp.Status
}

//Retrieve Data from SQL database
//Return struct containing db data
func retrieveData(db *sql.DB, reportId string) queryResult {
	var data queryResult
	res, err := db.Query("SELECT street,city,state,zipcode,azimuth,tilt,solrad_annual,ac_annual,reportId FROM OrderHistory Natural join NREL WHERE reportId=(?)", reportId)
	if err != nil {
		panic(err.Error())
	}
	for res.Next() {

		err = res.Scan(&data.street, &data.city, &data.state, &data.zipcode, &data.azimuth, &data.tilt, &data.solar_annual, &data.ac_annual, &data.reportId)

		if err != nil {
			panic(err.Error())
		}
	}
	return data
}

//Convert byte array into PNG format
//Return Base64Encoded PNG image in String format
func DisplayImage(imageData []byte) (string, error) {
	img, err := jpeg.Decode(bytes.NewReader(imageData))
	var buf bytes.Buffer
	if err != nil {
		log.Print("Unable to decode JPEG")
	}

	if err := png.Encode(&buf, img); err != nil {
		log.Print("ERROR")
	}

	imageData = buf.Bytes()
	return b64.StdEncoding.EncodeToString(imageData), err
}

//This function downloads the pdf for users
func downloadPDF(w http.ResponseWriter, r *http.Request, reportData []byte) {
	w.Header().Set("Content-Disposition", "attachment; filename=SampleReportTest.pdf")
	w.Header().Set("Content-Type", r.Header.Get("application/pdf"))
	w.Write(reportData)
}

//Display HTML page with data retrieved from DB for the advanced report
func DisplayPage(w http.ResponseWriter, r *http.Request) {
	//read data temporarly for testing
	session, _ := store.Get(r, "cookie-name")

	var reportId string = fmt.Sprint(session.Values["reportId"])
	var imageBaseStr [5]string
	var b2 []byte
	var b3 []byte
	var b4 []byte
	var b5 []byte
	var b6 []byte
	var data queryResult
	var tablres ExportEV
	var reportData []byte
	var jsonData []byte
	var err error
	token, _ := eagleViewToken()
	db, _ := connectDb()
	fmt.Printf("Report: %s\n", reportId)
	var waitgroup sync.WaitGroup
	waitgroup.Add(7)

	var url string
	data = retrieveData(db, reportId)

	go func() {
		defer waitgroup.Done()
		url, _ = downloadReport(token, reportId)
		getJsonFile(url, reportId)
	}()

	go func() {
		defer waitgroup.Done()
		b2, _ = getReportImage(token, reportId, 6)
		imageBaseStr[0], _ = DisplayImage(b2)
	}()

	go func() {
		defer waitgroup.Done()
		b3, _ = getReportImage(token, reportId, 22)
		imageBaseStr[1], _ = DisplayImage(b3)
	}()

	go func() {
		defer waitgroup.Done()
		b4, _ = getReportImage(token, reportId, 23)
		imageBaseStr[2], _ = DisplayImage(b4)
	}()

	go func() {
		defer waitgroup.Done()
		b5, _ = getReportImage(token, reportId, 24)
		imageBaseStr[3], _ = DisplayImage(b5)
	}()

	go func() {
		defer waitgroup.Done()
		b6, _ = getReportImage(token, reportId, 25)
		imageBaseStr[4], _ = DisplayImage(b6)
	}()

	go func() {
		defer waitgroup.Done()
		reportData, _, _ = getReportFile(token, reportId, 75, 2)
	}()

	waitgroup.Wait()
	waitgroup.Add(1)
	go func() {
		defer waitgroup.Done()
		if len(jsonData) == 0 {
			jsonData, _ = os.ReadFile("RadianceModel" + reportId + ".json")
			if err != nil {
				log.Println(err.Error())
			}
		}
	}()

	waitgroup.Wait()
	var tab ExportEV
	tablres = unmarshalJSON(jsonData, tab)
	if r.Method == http.MethodPost {
		downloadPDF(w, r, reportData)
	}

	var SIZE int = len(tablres.Roofs)
	var JsonRes = make([]ReportResult, SIZE)
	JsonRes = convertJsonToStruct(JsonRes, tablres, data)

	HomePageVars := PageVariables{ //store the date and time in a struct
		Address:    fmt.Sprintf("%s, %s, %s %s", data.street, data.city, data.state, data.zipcode),
		Ac_annual:  data.solar_annual * 365,
		ReportId:   data.reportId,
		TopImage:   imageBaseStr[0],
		NorthImage: imageBaseStr[1],
		SouthImage: imageBaseStr[2],
		EastImage:  imageBaseStr[3],
		WestImage:  imageBaseStr[4],
		JsonMes:    JsonRes,
	}

	t, err := template.ParseFiles("html/advanceReport.html")
	if err != nil { // if there is an error
		log.Print("template executing error: ", err) //log it
	}
	err = t.ExecuteTemplate(w, "html", HomePageVars) //execute the template and pass it the HomePageVars struct to fill in the gaps
	if err != nil {                                  // if there is an error
		log.Print("template executing error: ", err) //log it
	}
	//err = t.ExecuteTemplate(w, "table", &roofs)
	if err != nil { // if there is an error
		log.Print("template executing error: ", err) //log it
	}
}

//This function is where it asks user for payment for the report they are trying to place
func payment(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t, _ := template.ParseFiles("html/payment.html")
		t.Execute(w, nil)
	} else {
		r.ParseForm()
		token, _ := eagleViewToken()
		var paymentInput PaymentInfo
		paymentInput.CardNum = r.FormValue("cardnumber")
		paymentInput.ExpireMonth, _ = strconv.Atoi(r.FormValue("expmonth"))
		paymentInput.ExpireYear, _ = strconv.Atoi(r.FormValue("expyear"))
		/*Billing Input and Report Address*/
		var billInput Address
		billInput.FirstName = strings.ToLower(r.FormValue("firstname"))
		billInput.LastName = strings.ToLower(r.FormValue("lastname"))
		billInput.Street = strings.ToLower(r.FormValue("address"))
		billInput.City = strings.ToLower(r.FormValue("city"))
		billInput.State = strings.ToLower(r.FormValue("state"))
		billInput.Zip = strings.ToLower(r.FormValue("zip"))
		billInput.TypeRep = strings.ToLower(r.FormValue("Report Type"))
		billInput.Email = strings.ToLower(r.FormValue("email"))

		db, _ := connectDb()
		report := checkExistingOrder(db, billInput)

		//Credit card Validation Test
		card := creditcard.Card{Number: paymentInput.CardNum, Cvv: " ", Month: strconv.Itoa(paymentInput.ExpireMonth), Year: strconv.Itoa(paymentInput.ExpireYear)}
		err := card.Method()
		if err != nil {
			fmt.Print(err.Error())
		}
		var cid int = 0
		var ctype string = card.Company.Long
		switch ctype {
		case "MasterCard":
			cid = 3
		case "Visa":
			cid = 2
		case "Discover":
			cid = 4
		case "AmericanExpress":
			cid = 1
		}

		if cid == 0 {
			log.Println("This Credit Card Type is not Supported by Eagleview")

		} else {
			paymentInput.CardType = cid
		}

		if report.Valid {
			http.Redirect(w, r, "http://localhost:9090/formpage", http.StatusFound)
		} else {
			/*Placing Order Once Credit Card has been determined*/
			order := order(token, billInput, paymentInput, db)
			invoice := invoice(billInput, order)
			downloadPDF(w, r, invoice)
		}

		billInput.FirstName = ""
		billInput.LastName = ""
		paymentInput.CardNum = ""
		paymentInput.ExpireMonth = 0
		paymentInput.ExpireYear = 0
	}
}

//This function unmarshals JSON in to struct using threads
func unmarshalJSON(jsonData []byte, src ExportEV) ExportEV {
	var wg sync.WaitGroup

	out := make(chan []byte)
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func(w *sync.WaitGroup, in <-chan []byte) {
			defer w.Done()
			for input := range in {
				json.Unmarshal(input, &src)
			}
		}(&wg, out)

		out <- jsonData
	}
	close(out)
	wg.Wait()
	return src
}

//This function downloads the invoice for order on successful transaction
func invoice(billing Address, order OrderStats) []byte {
	var price int
	if billing.TypeRep == "Basic" {
		price = 75
	} else {
		price = 100
	}

	curentTime := time.Now()
	doc, _ := generator.New(generator.Invoice, &generator.Options{
		CurrencySymbol:  "$",
		TextTypeInvoice: "INVOICE",
		AutoPrint:       true,
	})

	doc.SetHeader(&generator.HeaderFooter{
		Pagination: true,
	})

	doc.SetFooter(&generator.HeaderFooter{
		Text:       "This Report # can be used to keep track of report",
		Pagination: true,
	})

	doc.SetRef("Ref: Report Invoice")

	doc.SetDescription("RenuLogix EagleView Report")
	doc.SetNotes(fmt.Sprintf("Report ID: %s\nThis Report # can be used to keep track of report", strconv.Itoa(order.ReportIds[0])))

	doc.SetDate(curentTime.Format("01-02-2006 Monday"))
	doc.SetPaymentTerm(curentTime.Format("01-02-2006 Monday"))

	logoBytes, _ := ioutil.ReadFile("./pics/RenuLogix-Logo.png")

	doc.SetCompany(&generator.Contact{
		Name: "Renulogix",
		Logo: &logoBytes,
		Address: &generator.Address{
			Address:    "85 N Raymond Ave",
			Address2:   "Pasadena, CA",
			PostalCode: "91103",
		},
	})

	doc.SetCustomer(&generator.Contact{
		Name: fmt.Sprintf("%s %s", billing.FirstName, billing.LastName),
		Address: &generator.Address{
			Address:    billing.Street,
			Address2:   fmt.Sprintf("%s , %s", billing.City, billing.State),
			PostalCode: billing.Zip,
		},
	})
	doc.AppendItem(&generator.Item{
		Name:     billing.TypeRep,
		UnitCost: strconv.Itoa(price),
		Quantity: "1",
	})
	pdf, err := doc.Build()
	if err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		fmt.Printf(err.Error())
	}
	var data []byte = buf.Bytes()

	return data
}

//This function Converts the json values from the report into struct
func convertJsonToStruct(test []ReportResult, tablres ExportEV, data queryResult) []ReportResult {
	for i, roof := range tablres.Roofs {
		test[i].Designator = roof.Designator
		test[i].Unroundedsize = roof.Unroundedsize
		test[i].Pitch = roof.Pitch
		test[i].PitchDeg = roof.PitchDeg
		test[i].Orientation = fmt.Sprintf("%.2f", roof.Orientation)
		test[i].Tsrf = fmt.Sprintf("%.2f", roof.Irradiance[0].Tsrf)
		test[i].Sa = fmt.Sprintf("%.2f", roof.Irradiance[0].Sa)
		test[i].SunHours = fmt.Sprintf("%.2f", roof.Irradiance[0].Tsrf*data.solar_annual*365)
	}
	return test
}

//This function checks whether the file available for select report id
func checkReport(TokenObject Token, reportId string) ([]byte, string) {
	var Endpoint = "https://webservices.eagleview.com/v3/Report/GetReport?reportId=" + reportId
	req, err := http.NewRequest("GET", Endpoint, nil)
	if err != nil {
		fmt.Printf("Error with request\n")
	}
	req.Header.Add("Host", "webservices.eagleview.com")
	req.Header.Add("Authorization", TokenObject.TokenType+" "+TokenObject.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err.Error())
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Print(err.Error())
	}
	return bodyBytes, resp.Status
}

//This function retrieves the url for json file from eagleview
func downloadReport(token Token, reportId string) (string, string) {
	var Endpoint = "https://webservices.eagleview.com/v1/reports/" + reportId + "/file-links"
	req, err := http.NewRequest("GET", Endpoint, nil)
	if err != nil {
		log.Print(err.Error())
	}
	req.Header.Add("Host", "webservices.eagleview.com")
	req.Header.Add("Authorization", token.TokenType+" "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Print(err.Error())
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)

	var links Link
	json.Unmarshal(bodyBytes, &links)
	for _, i := range links.Links {
		if i.FileType == "RadianceDeliverableJSON" {
			return i.Link, resp.Status
		}
	}
	if err != nil {
		log.Print(err.Error())
	}

	return "", resp.Status
}

//This function downloads the json file on to local machine
func getJsonFile(url string, reportId string) error {
	g := got.New()
	if _, err2 := os.Stat("RadianceModel" + reportId + ".json"); err2 == nil {
		return err2
	}
	err := g.Download(url, "RadianceModel"+reportId+".json")
	if err != nil {
		log.Print(err.Error())
	}
	return err
}

//Main Function where SSL certificate to be implemented to run a secure connection
func main() {
	serverMuxA := http.NewServeMux()
	serverMuxA.HandleFunc("/formpage", lookUpPage)
	serverMuxA.HandleFunc("/reportDisplay", DisplayPage)
	/*Server the http for payment and placing order*/

	serverMuxB := http.NewServeMux()
	serverMuxB.HandleFunc("/payment", payment)

	go func() {
		serverMuxA.Handle("/pics/", http.StripPrefix("/pics/", http.FileServer(http.Dir("pics"))))
		serverMuxA.Handle("/pics", http.FileServer(http.Dir("pics/")))
		serverMuxA.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
		serverMuxA.Handle("/css", http.FileServer(http.Dir("css/")))
		serverMuxA.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
		serverMuxA.Handle("/js", http.FileServer(http.Dir("js/")))
	}()

	go func() {
		serverMuxB.Handle("/pics/", http.StripPrefix("/pics/", http.FileServer(http.Dir("pics"))))
		serverMuxB.Handle("/pics", http.FileServer(http.Dir("pics/")))
		serverMuxB.Handle("/css/", http.StripPrefix("/css/", http.FileServer(http.Dir("css"))))
		serverMuxB.Handle("/css", http.FileServer(http.Dir("css/")))
		serverMuxB.Handle("/js/", http.StripPrefix("/js/", http.FileServer(http.Dir("js"))))
		serverMuxB.Handle("/js", http.FileServer(http.Dir("js/")))
	}()

	go http.ListenAndServe(":9090", serverMuxA)
	http.ListenAndServe(":8888", serverMuxB)
}
