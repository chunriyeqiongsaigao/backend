package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	dsn string = "root:20030414Wzc.@tcp(127.0.0.1:3306)/course"
	db  *gorm.DB
	err error
	//url     string = "https://www.prince-tech.club"
	message  map[string]int
	semester int = 4
)

func main() {
	// 初始化 map
	message = make(map[string]int)
	message["2021214048"] = 0

	db, err = gorm.Open(mysql.Open(dsn))
	if err != nil {
		fmt.Println(err)
	}

	r := gin.Default()
	// dongdong 12-11 第一次提交 3个API
	r.GET("/getCourseInfo", getCourseInfo)
	r.GET("/writeDiscussion", writeDiscussion)
	r.GET("/login", login)
	// 加载CA证书
	caCert, err := ioutil.ReadFile("./https/https.crt")
	if err != nil {
		panic(err)
	}

	// 创建证书池
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// 创建TLS配置
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,                         // 最低支持 TLS 1.2
		CurvePreferences:         []tls.CurveID{tls.CurveP256, tls.X25519}, // 支持的椭圆曲线算法
		PreferServerCipherSuites: true,                                     // 使用服务器端的加密套件优先
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, // 使用 ECDHE-RSA-AES128-GCM-SHA256 加密套件
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, // 可选：使用 ECDHE-RSA-AES256-GCM-SHA384 加密套件
		},
		RootCAs: caCertPool,
	}

	// 创建带有TLS配置的HTTP服务器
	server := &http.Server{
		Addr:      ":443",
		TLSConfig: tlsConfig,
		Handler:   r,
	}

	err = server.ListenAndServeTLS("./https/https.crt", "./https/https.key")
	if err != nil {
		panic(err)
	}

}

type Course struct {
	Id      int
	Name    string
	Time    int
	Summary string
}

func getCourseInfo(c *gin.Context) {
	name := c.Query("name")
	var course Course
	if err := db.Where("name = ?", name).Find(&course).Error; err != nil {
		c.JSON(400, gin.H{"error": err})
	}
	var discussions []Discussion
	if err := db.Where("course_name = ?", name).Order("time DESC").Find(&discussions).Error; err != nil {
		c.JSON(400, gin.H{"error": err})
	}

	for i, v := range discussions {
		discussions[i].Time = TimeAgo(v.Time)
	}

	var teachers []string
	if err := db.Table("relations").Select("name").Where("course_name = ?", name).Scan(&teachers).Error; err != nil {
		fmt.Println(err)
	}

	type CAD struct {
		Course     Course
		Teachers   []string
		Discussion []Discussion
	}
	var cad CAD
	cad.Course = course
	cad.Teachers = teachers
	cad.Discussion = discussions

	c.JSON(200, cad)
}

type Discussion struct {
	ID         int
	CourseName string
	Username   string
	Type       string
	Comment    string
	Time       string
}

func writeDiscussion(c *gin.Context) {
	var discussion Discussion
	discussion.Time = c.Query("time")
	discussion.Username = c.Query("username")
	discussion.Type = c.Query("type")
	discussion.CourseName = c.Query("name")
	discussion.Comment = c.Query("comment")
	if err := db.Create(&discussion).Error; err != nil {
		c.JSON(400, gin.H{"error": err})
	}
	c.Status(200)
}

type User struct {
	ID       int
	Username string
	Grade    int
	Type     string
}

func login(c *gin.Context) {
	username := c.Query("username")
	// 使用用户名进行查询
	var user User
	result := db.Where("username = ?", username).First(&user)
	if result.Error != nil {
		fmt.Println(result.Error)
	}

	// 返回用户信息
	c.JSON(200, user)
}

func getSemesterInfo(c *gin.Context) {
	semesterList := getSemesterList()
	c.JSON(200, semesterList)
}
