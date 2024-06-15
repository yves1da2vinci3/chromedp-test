package main

import (
	"bytes"
	"context"
	"html/template"
	"log"
	"net/url"
	"os"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
	"github.com/gofiber/fiber/v2"
)

type Company struct {
	Name    string `json:"name"`
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
	Zip     string `json:"zip"`
}

type Item struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Taxes    float64 `json:"taxes"`
	Price    float64 `json:"price"`
}

type Prices struct {
	Subtotal float64 `json:"subtotal"`
	Discount float64 `json:"discount"`
	Taxes    float64 `json:"taxes"`
	Total    float64 `json:"total"`
}

type EventData struct {
	FromCompany   Company `json:"fromCompany"`
	ToCompany     Company `json:"toCompany"`
	InvoiceNumber string  `json:"invoiceNumber"`
	IssueDate     string  `json:"issueDate"`
	DueDate       string  `json:"dueDate"`
	Items         []Item  `json:"items"`
	Prices        Prices  `json:"prices"`
	ShowTerms     bool    `json:"showTerms"`
}

func main() {
	app := fiber.New()

	app.Post("/generate-pdf", generatePDFHandler)

	log.Fatal(app.Listen(":3000"))
}

func generatePDFHandler(c *fiber.Ctx) error {
	// Parse the JSON body into EventData struct
	var eventData EventData
	if err := c.BodyParser(&eventData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid input",
		})
	}

	// Render the template with data
	htmlContent, err := renderTemplate("templates/eventReceipt.handlebars", eventData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to render template",
		})
	}

	// Generate PDF
	pdfBytes, err := generatePDF(htmlContent)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to generate PDF",
		})
	}

	// Set the correct content-type and return the PDF
	c.Set("Content-Type", "application/pdf")
	return c.Send(pdfBytes)
}

func renderTemplate(templatePath string, data interface{}) (string, error) {
	// Read template file
	tmplBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	// Parse template
	tmpl, err := template.New("template").Parse(string(tmplBytes))
	if err != nil {
		return "", err
	}

	// Execute template with data
	var rendered bytes.Buffer
	err = tmpl.Execute(&rendered, data)
	if err != nil {
		return "", err
	}

	return rendered.String(), nil
}

func generatePDF(htmlContent string) ([]byte, error) {
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	var buf []byte
	err := chromedp.Run(ctx, printToPDF(htmlContent, &buf))
	if err != nil {
		return nil, err
	}
	return buf, nil
}

func printToPDF(htmlContent string, res *[]byte) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate("data:text/html," + url.PathEscape(htmlContent)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			buf, _, err := page.PrintToPDF().
				WithPrintBackground(true).
				WithPaperWidth(8.27).   // A4 width in inches
				WithPaperHeight(11.69). // A4 height in inches
				Do(ctx)
			if err != nil {
				return err
			}
			*res = buf
			return nil
		}),
	}
}
