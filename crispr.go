package main

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/antonybholmes/go-jobs"
	awsjobs "github.com/antonybholmes/go-jobs/aws"
	awssys "github.com/antonybholmes/go-sys/aws"
	"github.com/antonybholmes/go-sys/log"
)

type (
	CrisprProcessor struct {
		s3Store            *awssys.S3Store
		s3JobStore         *awsjobs.S3JobStore
		jobCompletedSender *awsjobs.SQSJobStatusQueue
	}
)

const (
	CrisprPythonScript = "scripts/crispr.py"
)

func NewCrisprProcessor(s3Store *awssys.S3Store,
	s3JobStore *awsjobs.S3JobStore,
	jobCompletedSender *awsjobs.SQSJobStatusQueue) *CrisprProcessor {
	return &CrisprProcessor{
		s3Store:            s3Store,
		s3JobStore:         s3JobStore,
		jobCompletedSender: jobCompletedSender,
	}
}

func (p *CrisprProcessor) ProcessCrisprFile(ctx context.Context, job *jobs.Job) error {
	log.Info().Str("jobId", job.ID).Msg("Received job")

	if len(job.Files) == 0 {
		log.Error().Str("jobId", job.ID).Msg("No files in job")
		return nil
	}

	jobFile := job.Files[0]

	rc, err := p.s3JobStore.DownloadJobFile(ctx, jobFile)

	if err != nil {
		log.Error().Str("jobId", job.ID).Err(err).Msg("Failed to download job file")
		return err
	}

	// write file to local disk or process it as needed
	defer rc.Close()

	localPath, err := writeLocalFile(jobFile, rc)

	if err != nil {
		log.Error().Str("jobId", job.ID).Err(err).Msg("Failed to write local file")
		return err
	}

	// run shell command or any processing needed on the local file
	log.Info().Str("jobId", job.ID).Str("localPath", localPath).Msg("Processing local file")
	localPath, err = processLocalFile(job, localPath)
	if err != nil {
		log.Error().Str("jobId", job.ID).Err(err).Msg("Failed to process local file")
		return err
	}

	// load file so it can be uploaded to S3
	key := "processed_" + job.ID + ".txt"

	file, err := os.Open(localPath)
	if err != nil {
		log.Error().Str("jobId", job.ID).Err(err).Msg("Failed to open local file")
		return err
	}
	defer file.Close()

	// need file size for S3 upload
	stat, err := file.Stat()
	if err != nil {
		log.Error().Str("jobId", job.ID).Err(err).Msg("Failed to stat local file")
		return err
	}
	fileSize := stat.Size()

	// get basename e.g. output.txt
	filename := filepath.Base(localPath)

	// Make a key to save file
	key = awsjobs.MakeS3OutputKey(&job.BaseJob, filename)

	// Write the file to S3
	err = p.s3Store.Upload(ctx, key, awssys.ContentTypeTextPlain, file)

	if err != nil {
		log.Error().Str("jobId", job.ID).Err(err).Msg("Failed to upload file to S3")
		return err
	}

	// make new output job file
	outputFile := jobs.NewJobFile(jobs.JobFileTypeOutput,
		jobFile.Bucket,
		key,
		filename,
		awssys.ContentTypeTextPlain,
		fileSize,
		nil)

	// Send these files back to the job server so we
	// know its finished and job server can record them
	// and make output links etc
	files := []*jobs.JobFile{outputFile}

	// Once finished, notify the completion queue with the job status
	jobStatusItem := jobs.NewJobStatusQueueItem(job.ID,
		jobs.JobStatusCompleted,
		"",
		files)
	p.jobCompletedSender.SendJobStatus(ctx, jobStatusItem)

	// Clean up local files if needed
	os.Remove(localPath)

	return nil
}

func processLocalFile(job *jobs.Job, path string) (string, error) {
	// Implement the actual processing logic here
	// For now, just log the processing
	log.Info().Str("jobId", job.ID).Str("localPath", path).Msg("Processing local file")

	out := "/data/tmp/processed_" + job.ID + ".txt"

	// Run the shell command
	cmd := exec.Command("python", CrisprPythonScript, "--file="+path, "--out="+out)
	err := cmd.Run()
	if err != nil {
		log.Error().Str("jobId", job.ID).Err(err).Msg("Failed to run shell command")
		return "", err
	}

	return out, nil
}

func writeLocalFile(file *jobs.JobFile, rc io.ReadCloser) (string, error) {
	// key file name is unique so can strip it
	filename := filepath.Base(file.Key)

	path := "/data/tmp/" + filename

	localFile, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, rc)
	if err != nil {
		return "", err
	}

	return path, nil
}
