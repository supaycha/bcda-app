package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"

	"github.com/CMSgov/bcda-app/bcda/responseutils"

	log "github.com/sirupsen/logrus"

	"github.com/CMSgov/bcda-app/bcda/client"
	"github.com/CMSgov/bcda-app/bcda/database"
	"github.com/CMSgov/bcda-app/bcda/models"
	"github.com/bgentry/que-go"
)

var (
	qc *que.Client
)

type jobEnqueueArgs struct {
	ID             int
	AcoID          string
	UserID         string
	BeneficiaryIDs []string
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	filePath := os.Getenv("BCDA_WORKER_ERROR_LOG")
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.Info("Failed to log to file; using default stderr")
	}
}

func processJob(j *que.Job) error {
	log.Info("Worker started processing job ", j.ID)

	db := database.GetGORMDbConnection()
	defer db.Close()

	jobArgs := jobEnqueueArgs{}
	err := json.Unmarshal(j.Args, &jobArgs)
	if err != nil {
		return err
	}

	var exportJob models.Job
	err = db.First(&exportJob, "ID = ?", jobArgs.ID).Error
	if err != nil {
		return err
	}

	exportJob.Status = "In Progress"
	err = db.Save(exportJob).Error
	if err != nil {
		return err
	}

	bb, err := client.NewBlueButtonClient()
	if err != nil {
		log.Error(err)
		return err
	}

	err = writeEOBDataToFile(bb, jobArgs.AcoID, jobArgs.BeneficiaryIDs)

	if err != nil {
		exportJob.Status = "Failed"
	} else {
		exportJob.Status = "Completed"
	}

	err = db.Save(exportJob).Error
	if err != nil {
		return err
	}

	log.Info("Worker finished processing job ", j.ID)

	return nil
}

func writeEOBDataToFile(bb client.APIClient, acoID string, beneficiaryIDs []string) error {
	re := regexp.MustCompile("[a-fA-F0-9]{8}(?:-[a-fA-F0-9]{4}){3}-[a-fA-F0-9]{12}")
	if !re.Match([]byte(acoID)) {
		err := errors.New("Invalid ACO ID")
		log.Error(err)
		return err
	}

	if bb == nil {
		err := errors.New("Blue Button client is required")
		log.Error(err)
		return err
	}

	dataDir := os.Getenv("FHIR_PAYLOAD_DIR")
	f, err := os.Create(fmt.Sprintf("%s/%s.ndjson", dataDir, acoID))
	if err != nil {
		log.Error(err)
		return err
	}

	defer f.Close()

	w := bufio.NewWriter(f)

	for _, beneficiaryID := range beneficiaryIDs {
		pData, err := bb.GetExplanationOfBenefitData(beneficiaryID)
		if err != nil {
			log.Error(err)
			appendErrorToFile(acoID, responseutils.Exception, responseutils.BbErr, fmt.Sprintf("Error retrieving ExplanationOfBenefit for beneficiary %s in ACO %s", beneficiaryID, acoID))
		} else {
			fhirBundleToResourceNDJSON(w, pData, "ExplanationOfBenefits", beneficiaryID, acoID)
		}
	}

	w.Flush()

	return nil
}

func appendErrorToFile(acoID, code, detailsCode, detailsDisplay string) {
	oo := responseutils.CreateOpOutcome(responseutils.Error, code, detailsCode, detailsDisplay)

	dataDir := os.Getenv("FHIR_PAYLOAD_DIR")
	fileName := fmt.Sprintf("%s/%s-error.ndjson", dataDir, acoID)
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Error(err)
	}

	defer f.Close()

	ooBytes, err := json.Marshal(oo)
	if err != nil {
		log.Error(err)
	}

	if _, err = f.WriteString(string(ooBytes) + "\n"); err != nil {
		log.Error(err)
	}
}

func fhirBundleToResourceNDJSON(w *bufio.Writer, jsonData, jsonType, beneficiaryID, acoID string) {
	var jsonOBJ map[string]interface{}
	err := json.Unmarshal([]byte(jsonData), &jsonOBJ)
	if err != nil {
		log.Error(err)
		appendErrorToFile(acoID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error UnMarshaling %s from data for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID))
		return
	}

	entries := jsonOBJ["entry"]

	// There might be no entries.  If this happens we can't iterate over them.
	if entries != nil {

		for _, entry := range entries.([]interface{}) {
			entryJSON, err := json.Marshal(entry)
			// This is unlikely to happen because we just unmarshalled this data a few lines above.
			if err != nil {
				log.Error(err)
				appendErrorToFile(acoID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error Marshaling %s to Json for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID))
				continue
			}
			_, err = w.WriteString(string(entryJSON) + "\n")
			if err != nil {
				log.Error(err)
				appendErrorToFile(acoID, responseutils.Exception, responseutils.InternalErr, fmt.Sprintf("Error writing %s to file for beneficiary %s in ACO %s", jsonType, beneficiaryID, acoID))
			}
		}
	}

}

func waitForSig() {
	signalChan := make(chan os.Signal, 1)
	defer close(signalChan)

	signal.Notify(signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	exitChan := make(chan int)
	defer close(exitChan)

	go func() {
		for {
			s := <-signalChan
			switch s {
			case syscall.SIGINT:
				fmt.Println("interrupt")
				exitChan <- 0
			case syscall.SIGTERM:
				fmt.Println("force stop")
				exitChan <- 0
			case syscall.SIGQUIT:
				fmt.Println("stop and core dump")
				exitChan <- 0
			}
		}
	}()

	code := <-exitChan
	os.Exit(code)
}

func setupQueue() *pgx.ConnPool {
	queueDatabaseURL := os.Getenv("QUEUE_DATABASE_URL")
	pgxcfg, err := pgx.ParseURI(queueDatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	pgxpool, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:   pgxcfg,
		AfterConnect: que.PrepareStatements,
	})
	if err != nil {
		log.Fatal(err)
	}

	qc = que.NewClient(pgxpool)
	wm := que.WorkMap{
		"ProcessJob": processJob,
	}

	var workerPoolSize int
	if len(os.Getenv("WORKER_POOL_SIZE")) == 0 {
		workerPoolSize = 2
	} else {
		workerPoolSize, err = strconv.Atoi(os.Getenv("WORKER_POOL_SIZE"))
		if err != nil {
			log.Fatal(err)
		}
	}

	workers := que.NewWorkerPool(qc, wm, workerPoolSize)
	go workers.Start()

	return pgxpool
}

func main() {
	fmt.Println("Starting bcdaworker...")
	workerPool := setupQueue()
	defer workerPool.Close()
	waitForSig()
}
