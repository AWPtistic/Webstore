package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"go-webstore/db"
	"go-webstore/templates"

	"github.com/a-h/templ"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	etag "github.com/pablor21/echo-etag/v4"
)

var store = sessions.NewCookieStore([]byte("secret-key"))

func main() {
	e := echo.New()
	e.Use(etag.Etag())
	e.Static("assets", "./assets")

	database, err := db.Connect()
	if err != nil {
		log.Fatal("Failed to connect to the database")
	}
	defer database.Close()

	restrictAccess := func(requiredRole int) echo.MiddlewareFunc {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(ctx echo.Context) error {
				session, _ := store.Get(ctx.Request(), "session")
				role, ok := session.Values["role"].(int)
				if !ok || role < requiredRole {
					if !ok {
						return ctx.Redirect(http.StatusFound, "/?error=Must log in first")
					}
					return ctx.Redirect(http.StatusFound, "/?error=You are not authorized for that page!")
				}
				return next(ctx)
			}
		}
	}

	e.GET("/", func(ctx echo.Context) error {
		errorMsg := ctx.QueryParam("error")
		loginPage := `
		<!DOCTYPE html>
		<html>
		<head><title>Login</title></head>
		<body>
		<h1>Login</h1>
		<form method="POST" action="/login">
			<label>Email:</label>
			<input type="email" name="email" required>
			<label>Password:</label>
			<input type="password" name="password" required>
			<button type="submit">Login</button>
		</form>
		<p><a href="/store" style="text-decoration: underline; color: blue;">Continue as Guest</a></p>
		<p style="color: red;">%s</p>
		</body>
		</html>
		`
		return ctx.HTML(http.StatusOK, fmt.Sprintf(loginPage, errorMsg))
	})

	e.POST("/login", func(ctx echo.Context) error {
		email := ctx.FormValue("email")
		password := ctx.FormValue("password")

		user, err := db.AuthenticateUser(database, email, password)
		if err != nil || user.ID == 0 {
			return ctx.Redirect(http.StatusFound, "/?error=invalid user")
		}

		session, _ := store.Get(ctx.Request(), "session")
		session.Values["role"] = user.Role
		session.Values["name"] = user.FirstName
		session.Save(ctx.Request(), ctx.Response())

		if user.Role == 1 {
			return ctx.Redirect(http.StatusFound, "/order_entry")
		} else if user.Role == 2 {
			return ctx.Redirect(http.StatusFound, "/admin")
		}
		return ctx.Redirect(http.StatusFound, "/store")
	})

	e.GET("/logout", func(ctx echo.Context) error {
		session, _ := store.Get(ctx.Request(), "session")
		session.Options.MaxAge = -1
		session.Save(ctx.Request(), ctx.Response())
		return ctx.Redirect(http.StatusFound, "/")
	})

	e.GET("/store", func(ctx echo.Context) error {
		return Render(ctx, http.StatusOK, templates.Base())
	})

	e.POST("/purchase", func(ctx echo.Context) error {
		firstName := ctx.FormValue("firstName")
		lastName := ctx.FormValue("lastName")
		email := ctx.FormValue("email")
		product := ctx.FormValue("product")
		quantityStr := ctx.FormValue("quantity")
		donation := ctx.FormValue("donation")

		quantity, err := strconv.Atoi(quantityStr)
		if err != nil || quantity <= 0 {
			log.Println("Invalid quantity:", quantityStr)
			return ctx.String(http.StatusBadRequest, "Invalid quantity")
		}

		var price float64
		var productID int
		switch product {
		case "Family Friendly":
			price = 32000.0
			productID = 1
		case "Pure Sport":
			price = 150000.0
			productID = 2
		case "Budget Pick":
			price = 20000.0
			productID = 3
		default:
			log.Println("Invalid product:", product)
			return ctx.String(http.StatusBadRequest, "Invalid product selected")
		}

		subtotal := price * float64(quantity)
		tax := subtotal * 0.075
		total := subtotal + tax
		if donation == "yes" {
			total = float64(int(total + 1.00))
		}

		customer, err := db.GetCustomerByEmail(database, email)
		if err != nil {
			log.Println("Error fetching customer:", err)
			return ctx.String(http.StatusInternalServerError, "Error fetching customer")
		}
		if customer.ID == 0 {
			log.Printf("Customer not found. Adding new customer: %s", email)
			err = db.AddCustomer(database, firstName, lastName, email)
			if err != nil {
				log.Println("Error adding new customer:", err)
				return ctx.String(http.StatusInternalServerError, "Error adding customer")
			}
			customer, _ = db.GetCustomerByEmail(database, email)
		}

		err = db.AddOrder(database, customer.ID, productID, quantity, price, tax, 0.0)
		if err != nil {
			log.Println("Error adding new order:", err)
			return ctx.String(http.StatusInternalServerError, "Error placing order")
		}

		response := fmt.Sprintf("<p>Order submitted for: %s %s - %d %s(s) for a total of $%.2f</p>",
			firstName, lastName, quantity, product, total)
		return ctx.HTML(http.StatusOK, response)
	})

	e.GET("/dbQueries", func(ctx echo.Context) error {
		customers, err := db.GetAllCustomers(database)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Error fetching customers")
		}

		products, err := db.GetAllProducts(database)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Error fetching products")
		}

		orders, err := db.GetAllOrders(database)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Error fetching orders")
		}

		html := "<html><body><h1>Database Queries</h1>"
		html += "<h2>Customers</h2><table border='1'><tr><th>ID</th><th>First Name</th><th>Last Name</th><th>Email</th></tr>"
		for _, customer := range customers {
			html += fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%s</td></tr>", customer.ID, customer.FirstName, customer.LastName, customer.Email)
		}
		html += "</table>"

		html += "<h2>Products</h2><table border='1'><tr><th>ID</th><th>Name</th><th>Description</th><th>Price</th><th>In Stock</th></tr>"
		for _, product := range products {
			html += fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%s</td><td>%.2f</td><td>%d</td></tr>", product.ID, product.Name, product.Description, product.Price, product.InStock)
		}
		html += "</table>"

		html += "<h2>Orders</h2><table border='1'><tr><th>ID</th><th>Customer ID</th><th>Product ID</th><th>Quantity</th><th>Price</th><th>Total</th><th>Order Date</th></tr>"
		for _, order := range orders {
			html += fmt.Sprintf("<tr><td>%d</td><td>%d</td><td>%d</td><td>%d</td><td>%.2f</td><td>%.2f</td><td>%s</td></tr>", order.ID, order.CustomerID, order.ProductID, order.Quantity, order.Price, order.Total, order.OrderDate)
		}
		html += "</table></body></html>"

		return ctx.HTML(http.StatusOK, html)
	})

	e.GET("/admin", func(ctx echo.Context) error {
		customers, err := db.GetAllCustomers(database)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Error fetching customers")
		}

		products, err := db.GetAllProducts(database)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Error fetching products")
		}

		orders, err := db.GetAllOrdersWithDetails(database)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Error fetching orders")
		}

		tmplPath := filepath.Join("templates", "admin.html")
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Template loading error")
		}

		data := map[string]interface{}{
			"Customers": customers,
			"Products":  products,
			"Orders":    orders,
		}

		return tmpl.Execute(ctx.Response().Writer, data)
	}, restrictAccess(2))

	e.GET("/order_entry", func(ctx echo.Context) error {

		return Render(ctx, http.StatusOK, templates.OrderEntryPage())
	}, restrictAccess(1))

	e.GET("/get_product_quantity", func(ctx echo.Context) error {
		productName := ctx.QueryParam("product")
		if productName == "" {
			return ctx.String(http.StatusBadRequest, "Product name is required")
		}

		product, err := db.GetProductByName(database, productName)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Error fetching product quantity")
		}
		if product.ID == 0 {
			return ctx.String(http.StatusNotFound, "Product not found")
		}

		return ctx.String(http.StatusOK, fmt.Sprintf("%d", product.InStock))
	})

	e.GET("/get_customers", func(ctx echo.Context) error {
		nameQuery := ctx.QueryParam("name")
		if nameQuery == "" {
			return ctx.String(http.StatusBadRequest, "Name query parameter is required")
		}

		customers, err := db.GetCustomersByName(database, nameQuery)
		if err != nil {
			return ctx.String(http.StatusInternalServerError, "Error fetching customers")
		}

		if len(customers) == 0 {
			return ctx.HTML(http.StatusOK, "<tr><td colspan='3'>No matching customers found</td></tr>")
		}

		var response string
		for _, customer := range customers {
			response += fmt.Sprintf("<tr onclick='selectCustomer(this)'><td>%s</td><td>%s</td><td>%s</td></tr>",
				customer.FirstName, customer.LastName, customer.Email)
		}

		return ctx.HTML(http.StatusOK, response)
	})

	e.Logger.Fatal(e.Start(":8000"))
}

func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	if err := t.Render(ctx.Request().Context(), buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}
