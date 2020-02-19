package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dadosjusbr/remuneracao-magistrados/db"
	"github.com/dadosjusbr/storage"
	"github.com/joho/godotenv"

	"github.com/kelseyhightower/envconfig"
	"github.com/labstack/echo"
)

type config struct {
	Port   int    `envconfig:"PORT"`
	DBUrl  string `envconfig:"MONGODB_URI"`
	DBName string `envconfig:"MONGODB_NAME"`

	// StorageDB config
	SDBUri   string `envconfig:"SDB_URI"`
	SDBName  string `envconfig:"SDB_NAME"`
	SDBMiCol string `envconfig:"SDB_MICOL"`
	SDBAgCol string `envconfig:"SDB_AGCOL"`
}

var monthsLabelMap = map[int]string{
	1:  "Janeiro",
	2:  "Fevereiro",
	3:  "Março",
	4:  "Abril",
	5:  "Maio",
	6:  "Junho",
	7:  "Julho",
	8:  "Agosto",
	9:  "Setembro",
	10: "Outubro",
	11: "Novembro",
	12: "Dezembro",
}

var client *storage.Client

// TemplateRenderer is a custom html/template renderer for Echo framework
type TemplateRenderer struct {
	templates *template.Template
}

// Render renders a template document
func (t *TemplateRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

//SidebarElement contains the necessary info to render the sidebar
type SidebarElement struct {
	Label string
	URL   string
}

func getSidebarElements(dbClient *db.Client) ([]SidebarElement, error) {
	processedMonths, err := dbClient.GetProcessedMonths()
	if err != nil {
		return nil, fmt.Errorf("error retrieving all processed months from db --> %v", err)
	}

	var sidebarElements []SidebarElement

	for _, pm := range processedMonths {
		label := fmt.Sprintf("%s %d", monthsLabelMap[pm.Month], pm.Year)
		URL := fmt.Sprintf("/%d/%d", pm.Year, pm.Month)
		sidebarElements = append(sidebarElements, SidebarElement{Label: label, URL: URL})
	}

	return sidebarElements, nil
}

func handleMonthRequest(dbClient *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		month, err := strconv.Atoi(c.Param("month"))
		if err != nil {
			fmt.Println(fmt.Errorf("invalid month on the url: (%s) --> %v", c.Param("month"), err))
			return c.String(http.StatusBadRequest, "invalid month")
		}
		year, err := strconv.Atoi(c.Param("year"))
		if err != nil {
			fmt.Println(fmt.Errorf("invalid year on the url: (%s) --> %v", c.Param("year"), err))
			return c.String(http.StatusBadRequest, "invalid year")
		}

		monthResults, err := dbClient.GetMonthResults(month, year)
		if err != nil {
			if err == db.ErrDocNotFound {
				//TODO: render a 404 page
				fmt.Println("Document not found")
				return c.String(http.StatusNotFound, "not found")
			}
			fmt.Println(fmt.Errorf("unexpected error fetching month data from DB --> %v", err))
			return c.String(http.StatusInternalServerError, "unexpected error")
		}

		monthLabel := fmt.Sprintf("%s %d", monthsLabelMap[month], year)

		sidebarElements, err := getSidebarElements(dbClient)
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, "unexpected error")
		}

		viewModel := struct {
			Month           int
			Year            int
			MonthLabel      string
			SpreadsheetsURL string
			DatapackageURL  string
			SidebarElements []SidebarElement
			Statistics      []db.Statistic
		}{
			monthResults.Month,
			monthResults.Year,
			monthLabel,
			monthResults.SpreadsheetsURL,
			monthResults.DatapackageURL,
			sidebarElements,
			monthResults.Statistics,
		}
		return c.Render(http.StatusOK, "monthTemplate.html", viewModel)
	}
}

func handleMainPageRequest(dbClient *db.Client) echo.HandlerFunc {
	return func(c echo.Context) error {
		sidebarElements, err := getSidebarElements(dbClient)
		if err != nil {
			fmt.Println(err)
			return c.String(http.StatusInternalServerError, "unexpected error")
		}
		viewModel := struct {
			SidebarElements []SidebarElement
		}{
			sidebarElements,
		}
		return c.Render(http.StatusOK, "homePageTemplate.html", viewModel)
	}
}

// newClient Creates client to connect with DB and Cloud5
func newClient(c config) (*storage.Client, error) {
	db, err := storage.NewDBClient(c.SDBUri, c.SDBName, c.SDBMiCol, c.SDBAgCol)
	if err != nil {
		return nil, fmt.Errorf("error creating DB client: %q", err)
	}
	db.Collection(c.SDBMiCol)
	client, err := storage.NewClient(db, &storage.BackupClient{})
	if err != nil {
		return nil, fmt.Errorf("error creating storage.client: %q", err)
	}
	return client, nil
}

// type agencyTotalsYear struct {
// 	Year        int
// 	MonthTotals []monthTotals
// }

// type monthTotals struct {
// 	Month  int
// 	Wage   float64
// 	Perks  float64
// 	Others float64
// }

func getTotalsOfAgencyYear(c echo.Context) error {
	stateName := c.Param("estado")
	year, err := strconv.Atoi(c.Param("ano"))
	agencyName := c.Param("orgao")

	if err != nil {
		log.Fatal(err)
	}

	_, agenciesMonthlyInfo, err := client.GetDataForFirstScreen(stateName, year)

	var monthTotalsOfYear []monthTotals

	for _, agencyMonthlyInfo := range agenciesMonthlyInfo[agencyName] {
		monthTotals := monthTotals{agencyMonthlyInfo.Month, agencyMonthlyInfo.Summary.Wage.Total, agencyMonthlyInfo.Summary.Perks.Total, agencyMonthlyInfo.Summary.Others.Total}
		monthTotalsOfYear = append(monthTotalsOfYear, monthTotals)
	}

	if err != nil {
		log.Fatal(err)
	}

	agencyTotalsYear := agencyTotalsYear{year, monthTotalsOfYear}
	return c.JSON(http.StatusOK, agencyTotalsYear)
}

func getSummaryOfEntitiesOfState(c echo.Context) error {
	// employee1 := employee{"Marcos", 30000.0, 14000.0, 25000.0}
	// employee2 := employee{"Joeberth", 35000.0, 19000.0, 20000.0}
	// employee3 := employee{"Maria", 34000.0, 15000.0, 23000.0}
	// employees := []employee{employee1, employee2, employee3}
	// agencySummary := agencySummary{100, 250000.0, 100000.0, 26000.0}
	//agency1 := agency{"Tribunal de Justiça da Paraíba", "TJPB", "J", agencySummary, employees}
	state := state{"Paraíba", "pb", "url", nil}

	return c.JSON(http.StatusOK, state)
}

func getBasicInfoOfState(c echo.Context) error {
	stateName := c.Param("estado")
	agencies, _, err := client.GetDataForFirstScreen(stateName, 2019)
	if err != nil {
		log.Fatal(err)
	}

	var agenciesBasic []agencyBasic
	for k := range agencies {
		agenciesBasic = append(agenciesBasic, agencyBasic{agencies[k].ID, agencies[k].Entity})
	}

	state := state{stateName, "", "", agenciesBasic}

	return c.JSON(http.StatusOK, state)
}

func getSalaryOfAgencyMonthYear(c echo.Context) error {
	employee1 := employee{"Marcos", 30000.0, 14000.0, 25000.0}
	employee2 := employee{"Joeberth", 35000.0, 19000.0, 20000.0}
	employee3 := employee{"Maria", 34000.0, 15000.0, 23000.0}
	employees := []employee{employee1, employee2, employee3}
	return c.JSON(http.StatusOK, employees)
}

func getSummaryOfAgency(c echo.Context) error {
	agencySummary := agencySummary{100, 250000.0, 100000.0, 26000.0}
	return c.JSON(http.StatusOK, agencySummary)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}
	var conf config
	err := envconfig.Process("remuneracao-magistrados", &conf)
	if err != nil {
		log.Fatal(err.Error())
	}

	dbClient, err := db.NewClient(conf.DBUrl, conf.DBName)
	if err != nil {
		log.Fatal(err)
	}
	defer dbClient.CloseConnection()

	// Criando o client do storage
	client, err = newClient(conf)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Going to start listening at port:%d\n", conf.Port)

	e := echo.New()

	renderer := &TemplateRenderer{
		templates: template.Must(template.ParseGlob("templates/*.html")),
	}

	e.Renderer = renderer

	e.Static("/static", "templates/assets")

	e.GET("/", handleMainPageRequest(dbClient))
	e.GET("/:year/:month", handleMonthRequest(dbClient))

	// Return a summary of an entity. This information will be used in the head of the entity page.
	e.GET("/uiapi/v1/orgao/resumo/:orgao", getSummaryOfAgency)
	// Return all the salary of a month and year. This will be used in the point chart at the entity page.
	e.GET("/uiapi/v1/orgao/salario/:orgao/:ano/:mes", getSalaryOfAgencyMonthYear)
	// This will return information of a state and its entities and agencies. This will be used to provide basic information for the state page.
	e.GET("/uiapi/v1/entidades/resumo/:estado", getSummaryOfEntitiesOfState)
	// Return the total of salary of every month of a year of a agency. The salary is divided in Wage, Perks and Others. This will be used to plot the bars chart at the state page.
	e.GET("/uiapi/v1/orgao/totais/:estado/:orgao/:ano", getTotalsOfAgencyYear)
	// Return basic information of a state
	e.GET("/uiapi/v1/orgao/:estado", getBasicInfoOfState)

	s := &http.Server{
		Addr:         fmt.Sprintf(":%d", conf.Port),
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
	}
	e.Logger.Fatal(e.StartServer(s))
}

type state struct {
	Name      string
	ShortName string
	FlagURL   string
	Agency    []agencyBasic
}

type agencyBasic struct {
	Name           string
	AgencyCategory string
}

type agency struct {
	Name           string
	ShortName      string
	AgencyCategory string
	AgencySummary  agencySummary
	Employee       []employee
}

type employee struct {
	Name   string
	Wage   float64
	Perks  float64
	Others float64
}

type agencySummary struct {
	TotalEmployees int
	TotalWage      float64
	TotalPerks     float64
	MaxWage        float64
}

type agencyTotalsYear struct {
	Year        int
	MonthTotals []monthTotals
}

type monthTotals struct {
	Month  int
	Wage   float64
	Perks  float64
	Others float64
}
