package provider

import (
	"context"
	"net/http"
	"testing"

	"github.com/Bindtech-xyz/terraform-provider-coolify/internal/client"
)

// TestExplainS3StorageErrorNamesUnusableStorage locks in a follow-up to the
// coolify_s3_storage is_usable fix: a PUT .../backups 422 on s3_storage_uuid
// ("Select a usable S3 storage owned by your team.") gave no hint that the
// actual cause lives on the referenced coolify_s3_storage — this resource's
// own config can be entirely correct and it would still 422 forever if that
// storage's connection check failed. explainS3StorageError must look up the
// storage and name the real cause instead of relaying the raw message.
func TestExplainS3StorageErrorNamesUnusableStorage(t *testing.T) {
	c := newProviderTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"uuid":"s3x","is_usable":false}`))
	}))
	r := &volumeBackupResource{client: c}

	err := &client.Error{
		StatusCode: 422,
		Message:    "Validation failed.",
		Validation: map[string][]string{"s3_storage_uuid": {"Select a usable S3 storage owned by your team."}},
	}
	got := r.explainS3StorageError(context.Background(), err, "s3x")
	if got == err.Error() {
		t.Error("expected additional context naming the unusable storage, got the raw error unchanged")
	}
}

// TestExplainS3StorageErrorLeavesOtherErrorsAlone confirms the explanation
// only ever kicks in for the exact s3_storage_uuid validation case — any
// other error (network failure, a different validation field, a usable
// storage) passes through unchanged.
func TestExplainS3StorageErrorLeavesOtherErrorsAlone(t *testing.T) {
	r := &volumeBackupResource{client: nil}

	notA422 := &client.Error{StatusCode: 500, Message: "Internal Server Error"}
	if got := r.explainS3StorageError(context.Background(), notA422, "s3x"); got != notA422.Error() {
		t.Errorf("non-422 error must pass through unchanged, got %q", got)
	}

	otherField := &client.Error{
		StatusCode: 422,
		Message:    "Validation failed.",
		Validation: map[string][]string{"frequency": {"The frequency field is required."}},
	}
	if got := r.explainS3StorageError(context.Background(), otherField, "s3x"); got != otherField.Error() {
		t.Errorf("422 without an s3_storage_uuid field error must pass through unchanged, got %q", got)
	}

	if got := r.explainS3StorageError(context.Background(), otherField, ""); got != otherField.Error() {
		t.Errorf("empty s3StorageUUID must pass through unchanged, got %q", got)
	}
}
