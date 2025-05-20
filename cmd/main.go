package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Defines a "model" that we can use to communicate with the
// frontend or the database
// More on these "tags" like `bson:"_id,omitempty"`: https://go.dev/wiki/Well-known-struct-tags
// BookStore represents a book record in MongoDB and in JSON API responses.
type BookStore struct {
	MongoID     primitive.ObjectID `bson:"_id,omitempty" json:"-"`
	ID          string             `bson:"ID" form:"ID" json:"id"`
	BookName    string             `bson:"BookName" form:"BookName" json:"title"`
	BookAuthor  string             `bson:"BookAuthor" form:"BookAuthor" json:"author"`
	BookEdition string             `bson:"BookEdition,omitempty" form:"BookEdition" json:"edition,omitempty"`
	BookPages   string             `bson:"BookPages,omitempty" form:"BookPages" json:"pages,omitempty"`
	BookYear    string             `bson:"BookYear,omitempty" form:"BookYear" json:"year,omitempty"`
}

// Wraps the "Template" struct to associate a necessary method
// to determine the rendering procedure
type Template struct {
	tmpl *template.Template
}

// Preload the available templates for the view folder.
// This builds a local "database" of all available "blocks"
// to render upon request, i.e., replace the respective
// variable or expression.
// For more on templating, visit https://jinja.palletsprojects.com/en/3.0.x/templates/
// to get to know more about templating
// You can also read Golang's documentation on their templating
// https://pkg.go.dev/text/template
func loadTemplates() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

// Method definition of the required "Render" to be passed for the Rendering
// engine.
// Contraire to method declaration, such syntax defines methods for a given
// struct. "Interfaces" and "structs" can have methods associated with it.
// The difference lies that interfaces declare methods whether struct only
// implement them, i.e., only define them. Such differentiation is important
// for a compiler to ensure types provide implementations of such methods.
func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

// Here we make sure the connection to the database is correct and initial
// configurations exists. Otherwise, we create the proper database and collection
// we will store the data.
// To ensure correct management of the collection, we create a return a
// reference to the collection to always be used. Make sure if you create other
// files, that you pass the proper value to ensure communication with the
// database
// More on what bson means: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)

	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	coll := db.Collection(collecName)
	return coll, nil
}

// Here we prepare some fictional data and we insert it into the database
// the first time we connect to it. Otherwise, we check if it already exists.
func prepareData(client *mongo.Client, coll *mongo.Collection) {
	startData := []BookStore{
		{
			ID:          "example1",
			BookName:    "The Vortex",
			BookAuthor:  "José Eustasio Rivera",
			BookEdition: "958-30-0804-4",
			BookPages:   "292",
			BookYear:    "1924",
		},
		{
			ID:          "example2",
			BookName:    "Frankenstein",
			BookAuthor:  "Mary Shelley",
			BookEdition: "978-3-649-64609-9",
			BookPages:   "280",
			BookYear:    "1818",
		},
		{
			ID:          "example3",
			BookName:    "The Black Cat",
			BookAuthor:  "Edgar Allan Poe",
			BookEdition: "978-3-99168-238-7",
			BookPages:   "280",
			BookYear:    "1843",
		},
	}

	// This syntax helps us iterate over arrays. It behaves similar to Python
	// However, range always returns a tuple: (idx, elem). You can ignore the idx
	// by using _.
	// In the topic of function returns: sadly, there is no standard on return types from function. Most functions
	// return a tuple with (res, err), but this is not granted. Some functions
	// might return a ret value that includes res and the err, others might have
	// an out parameter.
	for _, book := range startData {
		cursor, err := coll.Find(context.TODO(), book)
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			panic(err)
		}
		if len(results) > 1 {
			log.Fatal("more records were found")
		} else if len(results) == 0 {
			result, err := coll.InsertOne(context.TODO(), book)
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("%+v\n", result)
			}

		} else {
			for _, res := range results {
				cursor.Decode(&res)
				fmt.Printf("%+v\n", res)
			}
		}
	}
}

// Generic method to perform "SELECT * FROM BOOKS" (if this was SQL, which
// it is not :D ), and then we convert it into an array of map. In Golang, you
// define a map by writing map[<key type>]<value type>{<key>:<value>}.
// interface{} is a special type in Golang, basically a wildcard...
// findAllBooks retrieves all books from the collection.
func findAllBooks(coll *mongo.Collection) []BookStore {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	if err != nil {
		panic(err)
	}
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}
	return results
}

func main() {
	// Connect to the database. Such defer keywords are used once the local
	// context returns; for this case, the local context is the main function
	// By user defer function, we make sure we don't leave connections
	// dangling despite the program crashing. Isn't this nice? :D
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: make sure to pass the proper username, password, and port
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://134.149.50.32:27017"))

	// This is another way to specify the call of a function. You can define inline
	// functions (or anonymous functions, similar to the behavior in Python)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// You can use such name for the database and collection, or come up with
	// one by yourself!
	coll, err := prepareDatabase(client, "exercise-1", "information")

	prepareData(client, coll)

	// Here we prepare the server
	e := echo.New()

	// Define our custom renderer
	e.Renderer = loadTemplates()

	// Log the requests. Please have a look at echo's documentation on more
	// middleware
	e.Use(middleware.Logger())

	e.Static("/css", "css")

	// Endpoint definition. Here, we divided into two groups: top-level routes
	// starting with /, which usually serve webpages. For our RESTful endpoints,
	// we prefix the route with /api to indicate more information or resources
	// are available under such route.
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", nil)
	})

	e.GET("/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.Render(200, "book-table", books)
	})

	// AUTHORS view
	e.GET("/authors", func(c echo.Context) error {
		cursor, err := coll.Find(context.TODO(), bson.D{})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
		}
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Cursor error"})
		}

		authorsMap := make(map[string]bool)
		var authors []string
		for _, book := range results {
			if !authorsMap[book.BookAuthor] {
				authorsMap[book.BookAuthor] = true
				authors = append(authors, book.BookAuthor)
			}
		}
		return c.Render(http.StatusOK, "authors", authors)
	})

	// YEARS view
	e.GET("/years", func(c echo.Context) error {
		cursor, err := coll.Find(context.TODO(), bson.D{})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Database error"})
		}
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Cursor error"})
		}

		yearsMap := make(map[string]bool)
		var years []string
		for _, book := range results {
			if !yearsMap[book.BookYear] {
				yearsMap[book.BookYear] = true
				years = append(years, book.BookYear)
			}
		}
		return c.Render(http.StatusOK, "years", years)
	})

	e.GET("/search", func(c echo.Context) error {
		return c.Render(200, "search-bar", nil)
	})

	e.GET("/create", func(c echo.Context) error {
		return c.Render(http.StatusOK, "create-form", nil)
	})

	// POST /api/books
	e.POST("/api/books", func(c echo.Context) error {
		var newBook BookStore
		if err := c.Bind(&newBook); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		}

		// Check for duplicate
		filter := bson.M{
			"ID":          newBook.ID,
			"BookName":    newBook.BookName,
			"BookAuthor":  newBook.BookAuthor,
			"BookEdition": newBook.BookEdition,
			"BookPages":   newBook.BookPages,
			"BookYear":    newBook.BookYear,
		}
		existing := coll.FindOne(context.TODO(), filter)
		if existing.Err() == nil {
			return c.JSON(http.StatusConflict, map[string]string{"error": "Book already exists"})
		}

		_, err := coll.InsertOne(context.TODO(), newBook)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not insert book"})
		}
		return c.JSON(http.StatusCreated, map[string]string{"status": "Book created"})
	})

	// PUT /api/books/:id
	e.PUT("/api/books/:id", func(c echo.Context) error {
		id := c.Param("id")
		var data map[string]interface{}
		if err := c.Bind(&data); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid update data"})
		}

		// Build BSON update document from allowed JSON fields
		updateFields := bson.M{}
		if v, ok := data["title"].(string); ok {
			updateFields["BookName"] = v
		}
		if v, ok := data["author"].(string); ok {
			updateFields["BookAuthor"] = v
		}
		if v, ok := data["edition"].(string); ok {
			updateFields["BookEdition"] = v
		}
		if v, ok := data["pages"].(string); ok {
			updateFields["BookPages"] = v
		}
		if v, ok := data["year"].(string); ok {
			updateFields["BookYear"] = v
		}
		if len(updateFields) == 0 {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "No valid fields to update"})
		}

		res, err := coll.UpdateOne(context.TODO(), bson.M{"ID": id}, bson.M{"$set": updateFields})
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Could not update book"})
		}
		if res.MatchedCount == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Book not found"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "Book updated"})
	})

	// DELETE /api/books/:id
	e.DELETE("/api/books/:id", func(c echo.Context) error {
		id := c.Param("id")
		filter := bson.M{"ID": id}
		res, err := coll.DeleteOne(context.TODO(), filter)
		if err != nil || res.DeletedCount == 0 {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Book not found or already deleted"})
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "Book deleted"})
	})

	// You will have to expand on the allowed methods for the path
	// `/api/route`, following the common standard.
	// A very good documentation is found here:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Methods
	// It specifies the expected returned codes for each type of request
	// method.
	e.GET("/api/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.JSON(http.StatusOK, books)
	})

	// We start the server and bind it to port 3030. For future references, this
	// is the application's port and not the external one. For this first exercise,
	// they could be the same if you use a Cloud Provider. If you use ngrok or similar,
	// they might differ.
	// In the submission website for this exercise, you will have to provide the internet-reachable
	// endpoint: http://<host>:<external-port>
	e.Logger.Fatal(e.Start(":3030"))
}
