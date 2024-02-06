package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

var privateKey *rsa.PrivateKey
var crt *x509.Certificate

type License struct {
	LicenseID          string    `json:"licenseId"`
	LicenseeName       string    `json:"licenseeName"`
	AssigneeName       string    `json:"assigneeName"`
	AssigneeEmail      string    `json:"assigneeEmail"`
	LicenseRestriction string    `json:"licenseRestriction"`
	CheckConcurrentUse bool      `json:"checkConcurrentUse"`
	Products           []Product `json:"products"`
	Metadata           string    `json:"metadata"`
	Hash               string    `json:"hash"`
	GracePeriodDays    int       `json:"gracePeriodDays"`
	AutoProlongated    bool      `json:"autoProlongated"`
	IsAutoProlongated  bool      `json:"isAutoProlongated"`
}

type Product struct {
	Code         string `json:"code"`
	FallbackDate string `json:"fallbackDate"`
	PaidUpTo     string `json:"paidUpTo"`
	Extended     bool   `json:"extended"`
}

func generateLicenseID() string {
	const allowedCharacters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const licenseLength = 10
	b := make([]byte, licenseLength)
	for i := range b {
		index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(allowedCharacters))))
		b[i] = allowedCharacters[index.Int64()]
	}
	return string(b)
}

func generateLicense(c *gin.Context) {

	var license License
	if err := c.ShouldBindJSON(&license); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	license.LicenseID = generateLicenseID()
	licenseStr, _ := json.Marshal(license)
	fmt.Printf("licenseStr:%s\n", licenseStr)
	// Sign the license using SHA1withRSA
	hashed := sha1.Sum(licenseStr)
	signature, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA1, hashed[:])

	licensePartBase64 := base64.StdEncoding.EncodeToString(licenseStr)
	signatureBase64 := base64.StdEncoding.EncodeToString(signature)
	crtBase64 := base64.StdEncoding.EncodeToString(crt.Raw)

	licenseResult := fmt.Sprintf("%s-%s-%s-%s", license.LicenseID, licensePartBase64, signatureBase64, crtBase64)
	fmt.Printf("licenseResult:%s\n", licenseResult)
	c.JSON(http.StatusOK, gin.H{"license": licenseResult})
}

func index(c *gin.Context) {
	c.HTML(http.StatusOK, "/index.html", gin.H{
		"title":        "请选择",
		"licenseeName": "Evaluator",
		"assigneeName": "Evaluator",
		"expiryDate":   "2099-12-31",
	})
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method
		origin := c.Request.Header.Get("Origin")

		if origin != "" {
			c.Header("Access-Control-Allow-Origin", "*") // 可将将 * 替换为指定的域名
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, UPDATE")
			c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
		}
		c.Next()
	}
}

func init() {
	// Load private key and certificate
	privateKeyPEM, err := os.ReadFile("./jetbra.key")
	if err != nil {
		panic("读取 jetbra.key 报错, 原因: " + err.Error())
	}

	block, _ := pem.Decode(privateKeyPEM)

	privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		panic("解析 jetbra.key 报错, 原因: " + err.Error())
	}

	crtPEM, err := os.ReadFile("./jetbra.pem")
	if err != nil {
		panic("读取 jetbra.pem 报错, 原因: " + err.Error())
	}
	block, _ = pem.Decode(crtPEM)
	crt, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic("解析 jetbra.pem 报错, 原因: " + err.Error())
	}
}

func main() {
	//初始化路由
	r := gin.Default()
	r.Use(cors())
	r.Static("static", "static")
	//加载模板
	r.LoadHTMLGlob("templates/*")
	r.GET("/", index)
	r.POST("/generateLicense", generateLicense)
	r.Run("0.0.0.0:8080")
}
