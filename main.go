package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/tealeg/xlsx"
)

type User struct {
	Number   int
	Id       int
	Name     string
	Email    string
	Phone    string
	City     string
	State    string
	Password string
}

func main() {

	db, err := sql.Open("postgres", "user=postgres password=123456 dbname=task sslmode=disable")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	router := gin.Default()

	router.Use(func(c *gin.Context) {

		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.GET("/excelsheet", func(c *gin.Context) {

		rows, err := db.Query("SELECT id, name, email, phone, city, state FROM employee")
		if err != nil {
			panic(err)
		}
		defer rows.Close()

		file := xlsx.NewFile()
		sheet, err := file.AddSheet("EmployeeData")
		if err != nil {
			panic(err)
		}

		rowIndex := 1

		for rows.Next() {

			var user User

			err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.Phone, &user.City, &user.State)
			if err != nil {
				panic(err)
			}
			row := sheet.AddRow()
			row.AddCell().SetValue(user.Id)
			row.AddCell().SetValue(user.Name)
			row.AddCell().SetValue(user.Email)
			row.AddCell().SetValue(user.Phone)
			row.AddCell().SetValue(user.City)
			row.AddCell().SetValue(user.State)
			rowIndex++
		}

		err = file.Save("EmployeeData.xlsx")
		if err != nil {
			panic(err)
		}

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", "EmployeeData.xlsx"))

		c.File("EmployeeData.xlsx")

	})

	router.POST("/upload", func(c *gin.Context) {

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = c.SaveUploadedFile(file, filepath.Join("./", file.Filename))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		xlFile, err := xlsx.OpenFile(file.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, row := range xlFile.Sheets[0].Rows {

			id, _ := row.Cells[0].Int()
			name := row.Cells[1].String()
			email := row.Cells[2].String()
			phone := row.Cells[3].String()
			city := row.Cells[4].String()
			state := row.Cells[5].String()

			var user User

			err := db.QueryRow("SELECT * FROM employee WHERE id = $1", id).Scan(&user.Id, &user.Name, &user.Email, &user.Phone, &user.City, &user.State)

			if err != nil {

				_, err := db.Exec("INSERT INTO employee (id, name, email, phone, city, state) VALUES ($1,$2,$3,$4,$5,$6)", id, name, email, phone, city, state)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
		}

		err = os.Remove(file.Filename)
		if err != nil {
			log.Println(err)
		}
		c.JSON(http.StatusOK, gin.H{"message": "Data added Successfully"})

	})

	router.POST("/users", func(c *gin.Context) {

		var user User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		result, err := db.Exec("INSERT INTO users (name, email, phone, city, state, password) VALUES ($1, $2, $3, $4, $5, $6)",
			user.Name, user.Email, user.Phone, user.City, user.State, user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data"})
			return
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get rows affected"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d row(s) affected", rowsAffected)})

	})

	router.GET("/api", func(c *gin.Context) {

		name := c.Query("name")
		password := c.Query("password")

		var count int

		err = db.QueryRow("SELECT COUNT(*) FROM users WHERE name = $1 AND password = $2", name, password).Scan(&count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		if count > 0 {
			c.JSON(http.StatusOK, gin.H{"message": "Login successful"})

		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "Invalid username or password"})

		}

	})

	router.GET("/users", func(c *gin.Context) {

		rows, err := db.Query("SELECT * FROM users")
		if err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"error": "Failed to get users",
			})
			return
		}
		defer rows.Close()

		users := []User{}
		for rows.Next() {
			var user User
			err := rows.Scan(&user.Id, &user.Name, &user.Email, &user.Phone, &user.City, &user.State, &user.Password)
			if err != nil {
				log.Fatal(err)
				c.JSON(500, gin.H{
					"error": "Failed to get users",
				})
				return
			}
			users = append(users, user)
		}
		if err := rows.Err(); err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"error": "Failed to get users",
			})
			return
		}

		c.JSON(200, gin.H{
			"users": users,
		})

	})

	router.DELETE("/delete", func(c *gin.Context) {

		file, err := c.FormFile("file1")

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = c.SaveUploadedFile(file, filepath.Join("./", file.Filename))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		xlFile, err := xlsx.OpenFile(file.Filename)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for _, row := range xlFile.Sheets[0].Rows {

			id, _ := row.Cells[0].Int()

			var user User

			err := db.QueryRow("SELECT * FROM employee where id=$1", id).Scan(&user.Id, &user.Name, &user.Email, &user.Phone, &user.City, &user.State)

			if err != nil {

				c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("user %v is not present", id)})
				continue
			}

			result, err := db.Exec("DELETE FROM employee WHERE id=$1", id)
			if err != nil {

				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			rowsAffected, err := result.RowsAffected()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get rows affected"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d row(s) Deleted", rowsAffected)})

		}

	})

	router.DELETE("/users/:id", func(c *gin.Context) {

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"error": "Invalid user ID",
			})
			return
		}

		_, err = db.Exec("DELETE FROM users WHERE id = $1", id)
		if err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"error": "Failed to delete user",
			})
			return
		}

		c.JSON(200, gin.H{
			"message": "User deleted successfully",
		})
	})

	router.PUT("/users/:id", func(c *gin.Context) {

		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"error": "Invalid user ID",
			})
			return
		}

		var update struct {
			Field string `json:"field"`
			Value string `json:"value"`
		}
		if err := c.ShouldBindJSON(&update); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var query string
		switch update.Field {
		case "name":
			query = "UPDATE users SET name = $1 WHERE id = $2"
		case "email":
			query = "UPDATE users SET email = $1 WHERE id = $2"
		case "phone":
			query = "UPDATE users SET phone = $1 WHERE id = $2"
		case "city":
			query = "UPDATE users SET city = $1 WHERE id = $2"
		case "state":
			query = "UPDATE users SET state = $1 WHERE id = $2"
		case "password":
			query = "UPDATE users SET password = $1 WHERE id = $2"
		default:
			c.JSON(400, gin.H{"error": "Invalid field name"})
			return
		}
		_, err = db.Exec(query, update.Value, id)
		if err != nil {
			log.Fatal(err)
			c.JSON(500, gin.H{
				"error": "Failed to update user",
			})
			return
		}

		c.JSON(200, gin.H{
			"message": "User updated successfully",
		})
	})

	router.Run(":8080")
}
