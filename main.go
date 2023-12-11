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
	// dongdong 12-11 第1次提交 3个API
	r.GET("/getCourseInfo", getCourseInfo)
	r.GET("/writeDiscussion", writeDiscussion)
	r.GET("/login", login)
	// dongdong 12-11 第2次提交 3个API
	r.GET("/getSemesterInfo", getSemesterInfo)
	r.GET("/getCourseInfoOfStudent", getCourseInfoOfStudent)
	r.GET("/getTeacher", getTeacher)

	// gbz 12-11 第一次提交3个API
	r.GET("/changeTeacher", changeTeacher)
    r.GET("/getCourseInfoOfTeacher", getCourseInfoOfTeacher)
    r.GET("/insert", insert)
	// gbz 12-11 第2次提交3个API
	r.GET("/getCourseInfoOfAdmin", getCourseInfoOfAdmin)
    r.GET("/getStudents", getStudents)
    r.GET("/getTeachers", getTeachers)
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

type Semester struct {
	Expand bool
	Time   int
	Title  string
	Course []string
}

func getSemesterList() []Semester {
	var courseList []Course
	if err := db.Order("time ASC").Find(&courseList).Error; err != nil {
		fmt.Println(err)
	}
	var semesterList []Semester
	for i := 0; i < 6; i++ {
		var semester Semester
		semester.Title = fmt.Sprintf("第%d学期", i+1)
		semester.Expand = true
		for _, v := range courseList {
			if v.Time == i+1 {
				semester.Course = append(semester.Course, v.Name)
				semester.Time = v.Time
			}

		}
		semesterList = append(semesterList, semester)
	}
	return semesterList
}

type Info struct {
	ID         int
	Username   string
	CourseName string
	Time       int
	Status     string
	Grade      string
	Teacher    string
}

func getCourseInfoOfStudent(c *gin.Context) {
	username := c.Query("username")
	time, _ := strconv.Atoi(c.Query("time"))
	semesterList := getSemesterList()

	var infos []Info
	if err := db.Where("username = ?", username).Order("time ASC").Find(&infos).Error; err != nil {
		fmt.Println(err)
	}

	type BigInfo struct {
		Semester Semester
		Info     []Info
	}

	var bigInfos []BigInfo
	for i, v := range semesterList {
		var item BigInfo
		item.Semester = v
		if i != time-1 {
			item.Semester.Expand = false
		}
		for _, n := range v.Course {
			for _, m := range infos {
				if m.CourseName == n {
					item.Info = append(item.Info, m)
				}
			}
		}
		bigInfos = append(bigInfos, item)
	}
	c.JSON(200, bigInfos)
}

type Relations struct {
	ID         int
	CourseName string
	Name       string
}

func getTeacher(c *gin.Context) {
	name := c.Query("name")
	var teachers []string
	if err := db.Table("relations").Select("name").Where("course_name = ?", name).Scan(&teachers).Error; err != nil {
		fmt.Println(err)
	}
	c.JSON(200, teachers)
}

func getTeacher(c *gin.Context) {
    name := c.Query("name")
    var teachers []string
    if err := db.Table("relations").Select("name").Where("course_name = ?", name).Scan(&teachers).Error; err != nil {
        fmt.Println(err)
    }
    c.JSON(200, teachers)
}

func changeTeacher(c *gin.Context) {
    name := c.Query("name")
    username := c.Query("username")
    teacher := c.Query("teacher")

    if err := db.Table("infos").Where("course_name = ? and username = ?", name, username).Update("teacher", teacher).Error; err != nil {
        fmt.Println(err)
        c.Status(400)
    }
    c.Status(200)
}

func getCourseInfoOfTeacher(c *gin.Context) {
    username := c.Query("username")
    time, _ := strconv.Atoi(c.Query("time"))
    var courses []string
    if err := db.Table("relations").Select("course_name").Where("name = ?", username).Scan(&courses).Error; err != nil {
        fmt.Println(err)
    }
    semesterList := getSemesterList()
    type BigInfo struct {
        Expand bool
        Title  string
        Name   string
        Status string
        Info   []Info
    }
    var bigInfos []BigInfo
    for i, v := range semesterList {
        var item BigInfo
        item.Title = fmt.Sprintf("第%d学期", i+1)
        if i == time-1 {
            item.Expand = true
            item.Status = "ing"
        } else if i < time-1 {
            item.Status = "done"
        } else {
            item.Status = "yet"
        }

        for _, j := range v.Course {
            for _, k := range courses {
                if j == k {
                    item.Name = k
                }
            }
        }

        var infos []Info
        if err := db.Table("infos").Where("course_name = ? and teacher = ?", item.Name, username).Find(&infos).Error; err != nil {
            fmt.Println(err)
            c.Status(400)
        }

        item.Info = infos

        bigInfos = append(bigInfos, item)
    }
    c.JSON(200, bigInfos)
}

func insert(c *gin.Context) {
    stime := 4
    var courses []Course
    if err := db.Find(&courses).Error; err != nil {
        fmt.Println(err)
    }
    type newCourse struct {
        CourseName string
        Time       int
        Status     string
    }
    var newCourseList []newCourse
    for _, v := range courses {
        var item newCourse
        item.CourseName = v.Name
        item.Time = v.Time
        if v.Time == stime {
            item.Status = "ing"
        } else if v.Time > stime {
            item.Status = "yet"
        } else {
            item.Status = "done"
        }

        newCourseList = append(newCourseList, item)
    }

    student := []string{"学生_1", "学生_2", "学生_3", "学生_4", "学生_5", "学生_6", "学生_7", "学生_8", "学生_9", "学生_10", "学生_11", "学生_12", "学生_13", "学生_14", "学生_15", "学生_16", "学生_17", "学生_18"}
    for _, username := range student {
        for _, course := range newCourseList {
            var teachers []string
            if err := db.Table("relations").Select("name").Where("course_name = ?", course.CourseName).Scan(&teachers).Error; err != nil {
                fmt.Println(err)
            }
            fmt.Println("teachers:", teachers)
            rand.Seed(time.Now().UnixNano()) // 设置随机种子

            random02 := rand.Intn(3)       // 生成 0 到 2 之间的随机数
            random15 := 85 + rand.Intn(15) // 生成 0 到 14 之间的随机数
            fmt.Println("random02:", random02)
            fmt.Println("random02:", random15)
            teacher := teachers[random02]
            fmt.Println()
            var info Info
            info.CourseName = course.CourseName
            info.Username = username
            info.Time = course.Time
            info.Status = course.Status
            info.Grade = fmt.Sprintf("%d", random15)
            info.Teacher = teacher
            fmt.Println(info)
            if err = db.Create(&info).Error; err != nil {
                fmt.Println(err)
            }
        }
    }
    c.Status(200)

}

func getCourseInfoOfAdmin(c *gin.Context) {
    semTime := 4
    //最终的结构体
    type Student struct {
        Status string
        Name   string
        Grade  string
    }
    type Teacher struct {
        Name     string
        Students []Student
    }
    type Course struct {
        Name     string
        Teachers []Teacher
    }
    type AdminInfo struct {
        Title   string
        Status  string
        Expand  bool
        Courses []Course
    }
    var admin_list []AdminInfo
    //得到学期课表
    semesterList := getSemesterList()
    //根据每个课拿到老师
    //根据老师拿到每个班的同学
    var item_admin AdminInfo
    for _, semester := range semesterList {
        var course_list []Course
        for _, course := range semester.Course {
            var item_course Course
            item_course.Name = course
            // 拿到课程的老师
            var teachers []string
            if err := db.Table("relations").Select("name").Where("course_name = ?", course).Scan(&teachers).Error; err != nil {
                fmt.Println(err)
            }
            // 拿到老师的学生
            var teachers_list []Teacher
            for _, teacher := range teachers {
                var item_teacher Teacher
                var infos []Info
                if err := db.Where("course_name = ? and teacher = ?", course, teacher).Find(&infos).Error; err != nil {
                    fmt.Println(err)
                }
                var students []Student
                for _, student := range infos {
                    var item_student Student
                    item_student.Name = student.Username
                    item_student.Grade = student.Grade
                    if semTime == student.Time {
                        item_student.Status = "ing"
                    } else if semTime < student.Time {
                        item_student.Status = "yet"
                    } else {
                        item_student.Status = "done"
                    }
                    students = append(students, item_student)
                }
             

func TimeAgo(millisecondsStr string) string {
    milliseconds, err := strconv.ParseInt(millisecondsStr, 10, 64)
    if err != nil {
        fmt.Println("解析毫秒数出错:", err)
        return ""
    }
    currentTime := time.Now()
    providedTime := time.Unix(milliseconds/1000, 0)

    duration := currentTime.Sub(providedTime)

    if duration < time.Hour {
        if int(duration.Minutes()) == 0 {
            return "刚刚"
        } else {
            return fmt.Sprintf("%d分钟前", int(duration.Minutes()))
        }
    } else if duration < time.Hour*24 {
        return fmt.Sprintf("%d小时前", int(duration.Hours()))
    } else if duration < time.Hour*24*30 {
        return fmt.Sprintf("%d天前", int(duration.Hours()/24))
    } else {
        months := int(duration.Hours() / (24 * 30))
        if months == 1 {
            return "1个月前"
        } else {
            return fmt.Sprintf("%d个月前", months)
        }
    }
}

func getStudents(c *gin.Context) {
    var students []string
    if err := db.Table("users").Where("type = ?","student").Select("username").Scan(&students).Error; err != nil {
        fmt.Println(err)
    }
    c.JSON(200,students)
}

func getTeachers(c *gin.Context) {
    var teachers []string
    if err := db.Table("users").Where("type = ?","teacher").Select("username").Scan(&teachers).Error; err != nil {
        fmt.Println(err)
    }
    c.JSON(200,teachers)
}
