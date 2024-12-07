package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func Connect() (*sql.DB, error) {
	connStr := "user=postgres dbname=go_webstore password=1453 host=localhost port=5432 sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("Database connection failed: ", err)
		return nil, err
	}

	log.Println("Successfully connected to the database")
	return db, nil
}

// Structs
type User struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
	Role      int
}

type Customer struct {
	ID        int
	FirstName string
	LastName  string
	Email     string
}

type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
	InStock     int
}

type Order struct {
	ID         int
	CustomerID int
	ProductID  int
	Quantity   int
	Price      float64
	Tax        float64
	Donation   float64
	Total      float64
	OrderDate  string
}

type OrderDetail struct {
	ID                int
	CustomerFirstName string
	CustomerLastName  string
	ProductName       string
	Quantity          int
	Price             float64
	Tax               float64
	Donation          float64
	Total             float64
	OrderDate         string
}

// Functions
func AuthenticateUser(db *sql.DB, email, password string) (User, error) {
	var user User
	query := `SELECT id, first_name, last_name, email, role FROM users WHERE email = $1 AND password = $2`
	err := db.QueryRow(query, email, password).Scan(&user.ID, &user.FirstName, &user.LastName, &user.Email, &user.Role)
	if err == sql.ErrNoRows {
		return User{}, nil // No user found
	}
	if err != nil {
		log.Printf("Error authenticating user: %v", err)
		return User{}, err
	}
	return user, nil
}

// Customers
func GetCustomerByEmail(db *sql.DB, email string) (Customer, error) {
	var customer Customer
	err := db.QueryRow("SELECT id, first_name, last_name, email FROM customers WHERE email = $1", email).
		Scan(&customer.ID, &customer.FirstName, &customer.LastName, &customer.Email)
	if err == sql.ErrNoRows {
		return Customer{}, nil
	}
	if err != nil {
		log.Println("Error fetching customer by email: ", err)
		return Customer{}, err
	}
	return customer, nil
}

func AddCustomer(db *sql.DB, firstName, lastName, email string) error {
	_, err := db.Exec(
		"INSERT INTO customers (first_name, last_name, email) VALUES ($1, $2, $3)",
		firstName, lastName, email,
	)
	if err != nil {
		log.Println("Error adding new customer: ", err)
		return err
	}
	return nil
}

func GetCustomersByName(db *sql.DB, nameQuery string) ([]Customer, error) {
	rows, err := db.Query(
		"SELECT id, first_name, last_name, email FROM customers WHERE first_name ILIKE $1 OR last_name ILIKE $1",
		nameQuery+"%",
	)
	if err != nil {
		log.Println("Error querying customers by name: ", err)
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Email); err != nil {
			log.Println("Error scanning customer row: ", err)
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}

func GetAllCustomers(db *sql.DB) ([]Customer, error) {
	rows, err := db.Query("SELECT id, first_name, last_name, email FROM customers")
	if err != nil {
		log.Println("Error querying customers: ", err)
		return nil, err
	}
	defer rows.Close()

	var customers []Customer
	for rows.Next() {
		var c Customer
		if err := rows.Scan(&c.ID, &c.FirstName, &c.LastName, &c.Email); err != nil {
			log.Println("Error scanning customer row: ", err)
			return nil, err
		}
		customers = append(customers, c)
	}
	return customers, nil
}

// Products
func GetProductByName(db *sql.DB, name string) (Product, error) {
	var product Product
	err := db.QueryRow(
		"SELECT id, name, description, price, in_stock FROM products WHERE name = $1", name,
	).Scan(&product.ID, &product.Name, &product.Description, &product.Price, &product.InStock)
	if err == sql.ErrNoRows {
		return Product{}, nil
	}
	if err != nil {
		log.Println("Error fetching product by name: ", err)
		return Product{}, err
	}
	return product, nil
}

func GetAllProducts(db *sql.DB) ([]Product, error) {
	rows, err := db.Query("SELECT id, name, description, price, in_stock FROM products")
	if err != nil {
		log.Println("Error querying products: ", err)
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price, &p.InStock); err != nil {
			log.Println("Error scanning product row: ", err)
			return nil, err
		}
		products = append(products, p)
	}
	return products, nil
}

// Orders
func AddOrder(db *sql.DB, customerID, productID, quantity int, price, tax, donation float64) error {
	_, err := db.Exec(
		"INSERT INTO orders (customer_id, product_id, quantity, price, tax, donation) VALUES ($1, $2, $3, $4, $5, $6)",
		customerID, productID, quantity, price, tax, donation,
	)
	if err != nil {
		log.Println("Error adding new order: ", err)
		return err
	}
	return nil
}

func GetAllOrders(db *sql.DB) ([]Order, error) {
	rows, err := db.Query("SELECT id, customer_id, product_id, quantity, price, tax, donation, total, order_date FROM orders")
	if err != nil {
		log.Println("Error querying orders: ", err)
		return nil, err
	}
	defer rows.Close()

	var orders []Order
	for rows.Next() {
		var o Order
		if err := rows.Scan(
			&o.ID, &o.CustomerID, &o.ProductID, &o.Quantity,
			&o.Price, &o.Tax, &o.Donation, &o.Total, &o.OrderDate,
		); err != nil {
			log.Println("Error scanning order row: ", err)
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func GetAllOrdersWithDetails(db *sql.DB) ([]OrderDetail, error) {
	rows, err := db.Query(`
		SELECT 
			o.id, 
			c.first_name, c.last_name,
			p.name, 
			o.quantity, o.price, o.tax, o.donation, o.total, o.order_date
		FROM orders o
		INNER JOIN customers c ON o.customer_id = c.id
		INNER JOIN products p ON o.product_id = p.id
	`)
	if err != nil {
		log.Println("Error querying order details: ", err)
		return nil, err
	}
	defer rows.Close()

	var orders []OrderDetail
	for rows.Next() {
		var order OrderDetail
		if err := rows.Scan(
			&order.ID, &order.CustomerFirstName, &order.CustomerLastName,
			&order.ProductName, &order.Quantity, &order.Price,
			&order.Tax, &order.Donation, &order.Total, &order.OrderDate,
		); err != nil {
			log.Println("Error scanning order detail row: ", err)
			return nil, err
		}
		 orders = append(orders, order)
	 }
	 return orders, nil
 }
