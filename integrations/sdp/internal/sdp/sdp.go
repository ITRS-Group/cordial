/*
Copyright © 2026 ITRS Group

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.

You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sdp

// ErrorResponse represents a typical error response from SDP v3 API.
type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ProblemAttributes represents the payload to create a new problem in SDP v3 API.
type ProblemAttributes struct {
	ID                 int64                     `json:"id,string,omitzero"`
	Title              string                    `json:"title,omitempty"`
	Description        string                    `json:"description"`
	ReportedTime       *DateTime                 `json:"reported_time,omitempty"`
	DueByTime          *DateTime                 `json:"due_by_time,omitempty"`
	ClosedTime         *DateTime                 `json:"closed_time,omitempty"`
	ReportedBy         *User                     `json:"reported_by"`
	Category           *NameID                   `json:"category,omitempty"`
	Impact             *NameID                   `json:"impact,omitempty"`
	Priority           *NameID                   `json:"priority,omitempty"`
	Subcategory        *NameID                   `json:"subcategory,omitempty"`
	Item               *NameID                   `json:"item,omitempty"`
	Urgency            *NameID                   `json:"urgency,omitempty"`
	Site               *NameID                   `json:"site,omitempty"`
	Group              *NameID                   `json:"group,omitempty"`
	Technician         *User                     `json:"technician,omitempty"`
	Status             *NameID                   `json:"status,omitempty"`
	Assets             []NameID                  `json:"assets,omitempty"`
	Services           []NameID                  `json:"services,omitempty"`
	UDFFields          map[string]any            `json:"udf_fields,omitempty"`
	Attachments        []NameID                  `json:"attachments,omitempty"`
	Template           *NameID                   `json:"template,omitempty"`
	ConfigurationItems []NameID                  `json:"configuration_items,omitempty"`
	ImpactDetails      *ProblemImpactDetails     `json:"impact_details,omitempty"`
	RootCause          *ProblemRootCause         `json:"root_cause,omitempty"`
	Symptoms           *ProblemSymptoms          `json:"symptoms,omitempty"`
	KnownErrorDetails  *ProblemKnownError        `json:"known_error_details,omitempty"`
	CloseDetails       *ProblemCloseDetails      `json:"close_details,omitempty"`
	ResolutionDetails  *ProblemResolutionDetails `json:"resolution_details,omitempty"`
	WorkAroundDetails  *ProblemWorkAroundDetails `json:"workaround_details,omitempty"`
	DisplayID          int64                     `json:"display_id,string,omitzero"`
	NotesPresent       bool                      `json:"notes_present,omitempty"`
	UpdatedTime        *DateTime                 `json:"updated_time,omitempty"`
	Lifecycle          *NameID                   `json:"lifecycle,omitempty"`
}

type ProblemImpactDetails struct {
	ImpactDetailsDescription string    `json:"impact_details_description,omitempty"`
	ImpactDetailsUpdatedTime *DateTime `json:"impact_details_updated_time,omitempty"`
	ImpactDetailsUpdatedBy   *User     `json:"impact_details_updated_by,omitempty"`
}

type ProblemRootCause struct {
	RootCauseDescription string    `json:"root_cause_description,omitempty"`
	RootCauseUpdatedOn   *DateTime `json:"root_cause_updated_on,omitempty"`
	RootCauseUpdatedBy   *User     `json:"root_cause_updated_by,omitempty"`
}

type ProblemSymptoms struct {
	SymptomsDescription string    `json:"symptoms_description,omitempty"`
	SymptomsUpdatedOn   *DateTime `json:"symptoms_updated_on,omitempty"`
	SymptomsUpdatedBy   *User     `json:"symptoms_updated_by,omitempty"`
}

type ProblemKnownError struct {
	KnownErrorComments         string    `json:"known_error_comments,omitempty"`
	IsKnownError               bool      `json:"is_known_error,omitempty"`
	KnownErrorDetailsUpdatedOn *DateTime `json:"known_error_details_updated_on,omitempty"`
	KnownErrorDetailsUpdatedBy *User     `json:"known_error_details_updated_by,omitempty"`
}

type ProblemCloseDetails struct {
	CloseDetailsComments  string    `json:"close_details_comments,omitempty"`
	ClosureCode           *NameID   `json:"closure_code,omitempty"`
	CloseDetailsUpdatedOn *DateTime `json:"close_details_updated_on,omitempty"`
	CloseDetailsUpdatedBy *User     `json:"close_details_updated_by,omitempty"`
}

type ProblemResolutionDetails struct {
	ResolutionDetailsDescription string    `json:"resolution_details_description,omitempty"`
	ResolutionDetailsUpdatedOn   *DateTime `json:"resolution_details_updated_on,omitempty"`
	ResolutionDetailsUpdatedBy   *User     `json:"resolution_details_updated_by,omitempty"`
}

type ProblemWorkAroundDetails struct {
	WorkAroundDetailsDescription string    `json:"workaround_details_description,omitempty"`
	WorkAroundDetailsUpdatedOn   *DateTime `json:"workaround_details_updated_on,omitempty"`
	WorkAroundDetailsUpdatedBy   *User     `json:"workaround_details_updated_by,omitempty"`
}

// RequestAttributes represents the payload to create a new ticket in SDP v3 API.
type RequestAttributes struct {
	ID                     int64               `json:"id,string,omitzero"`
	Subject                string              `json:"subject,omitempty"`
	Description            string              `json:"description,omitempty"`
	ImpactDetails          string              `json:"impact_details,omitempty"`
	EMailIDsToNotify       string              `json:"email_ids_to_notify,omitempty"`
	DeletePreTemplateTasks bool                `json:"delete_pre_template_tasks,omitempty"`
	UpdateReason           string              `json:"update_reason,omitempty"`
	CreatedTime            *DateTime           `json:"created_time,omitempty"`
	DueByTime              *DateTime           `json:"due_by_time,omitempty"`
	FirstResponseDueByTime *DateTime           `json:"first_response_due_by_time,omitempty"`
	IsFCR                  bool                `json:"is_fcr,omitempty"`
	StatusChangeComments   string              `json:"status_change_comments,omitempty"`
	ScheduledStartTime     *DateTime           `json:"scheduled_start_time,omitempty"`
	ScheduledEndTime       *DateTime           `json:"scheduled_end_time,omitempty"`
	Impact                 *NameID             `json:"impact,omitempty"`
	Status                 *NameID             `json:"status,omitempty"`
	Requester              *User               `json:"requester,omitempty"`
	Mode                   *NameID             `json:"mode,omitempty"`
	Site                   *NameID             `json:"site,omitempty"`
	Template               *NameID             `json:"template,omitempty"`
	SLA                    *NameID             `json:"sla,omitempty"`
	ServiceCategory        *NameID             `json:"service_category,omitempty"`
	Group                  *NameID             `json:"group,omitempty"`
	Technician             *User               `json:"technician,omitempty"`
	Priority               *NameID             `json:"priority,omitempty"`
	Level                  *NameID             `json:"level,omitempty"`
	Category               *NameID             `json:"category,omitempty"`
	Subcategory            *NameID             `json:"subcategory,omitempty"`
	Item                   *NameID             `json:"item,omitempty"`
	Urgency                *NameID             `json:"urgency,omitempty"`
	RequestType            *NameID             `json:"request_type,omitempty"`
	Assets                 []NameID            `json:"assets,omitempty"`
	UDFFields              map[string]any      `json:"udf_fields,omitempty"`
	Attachments            []RequestAttachment `json:"attachments,omitempty"`
	Resources              *RequestResources   `json:"resources,omitempty"`
	OnBehalfOf             *User               `json:"on_behalf_of,omitempty"`
	ConfigurationItems     []NameID            `json:"configuration_items,omitempty"`
	Editor                 *User               `json:"editor,omitempty"`
	Resolution             *RequestResolution  `json:"resolution,omitempty"`
	ServiceApprovers       []ServiceApprover   `json:"service_approvers,omitempty"`
	OnHoldScheduler        *OnHoldScheduler    `json:"on_hold_scheduler,omitempty"`
	ClosureInfo            *ClosureInfo        `json:"closure_info,omitempty"`
	LinkedToRequest        *RequestRequestID   `json:"linked_to_request,omitempty"`

	// The fields below are READ ONLY

	TimeElapsed               int64               `json:"time_elapsed,string,omitzero"`
	EMailCC                   string              `json:"email_cc,omitempty"`
	EMailTo                   string              `json:"email_to,omitempty"`
	EMailBCC                  string              `json:"email_bcc,omitempty"`
	CompletedByDenial         bool                `json:"completed_by_denial,omitempty"`
	CancellationRequested     bool                `json:"cancellation_requested,omitempty"`
	CancelFlagComments        *CancelFlagComments `json:"cancel_flag_comments,omitempty"`
	DisplayID                 int64               `json:"display_id,string,omitzero"`
	CreatedBy                 *User               `json:"created_by,omitempty"`
	RespondedTime             *DateTime           `json:"responded_time,omitempty"`
	CompletedTime             *DateTime           `json:"completed_time,omitempty"`
	Department                *NameID             `json:"department,omitempty"`
	DeletedTime               *DateTime           `json:"deleted_time,omitempty"`
	ServiceCost               float64             `json:"service_cost,omitempty"`
	TotalCost                 float64             `json:"total_cost,omitempty"`
	Lifecycle                 *NameID             `json:"lifecycle,omitempty"`
	IsServiceRequest          bool                `json:"is_service_request,omitempty"`
	ResolvedTime              *DateTime           `json:"resolved_time,omitempty"`
	LastUpdateTime            *DateTime           `json:"last_update_time,omitempty"`
	IsOverDue                 bool                `json:"is_over_due,omitempty"`
	IsFirstResponseOverDue    bool                `json:"is_first_response_over_due,omitempty"`
	IsEscalated               bool                `json:"is_escalated,omitempty"`
	IsRead                    bool                `json:"is_read,omitempty"`
	NotificationStatus        string              `json:"notification_status,omitempty"`
	ApprovalStatus            *NameID             `json:"approval_status,omitempty"`
	AssignedTime              *DateTime           `json:"assigned_time,omitempty"`
	DeletedAssets             string              `json:"deleted_assets,omitempty"`
	HasAttachments            bool                `json:"has_attachments,omitempty"`
	HasNotes                  bool                `json:"has_notes,omitempty"`
	HasLinkedRequests         bool                `json:"has_linked_requests,omitempty"`
	HasProject                bool                `json:"has_project,omitempty"`
	HasProblem                bool                `json:"has_problem,omitempty"`
	HasRequestInitiatedChange bool                `json:"has_request_initiated_change,omitempty"`
	HasChangeInitiatedRequest bool                `json:"has_change_initiated_request,omitempty"`
	HasPurchaseOrders         bool                `json:"has_purchase_orders,omitempty"`
	HasDraft                  bool                `json:"has_draft,omitempty"`
	IsReopened                bool                `json:"is_reopened,omitempty"`
	IsTrashed                 bool                `json:"is_trashed,omitempty"`
	UnrepliedCount            int64               `json:"unreplied_count,string,omitzero"`
	EditorStatus              int                 `json:"editor_status,omitempty"`
}

type OnHoldScheduler struct {
	ScheduledTime  *DateTime `json:"scheduled_time,omitempty"`
	Comments       string    `json:"comments,omitempty"`
	ChangeToStatus *NameID   `json:"change_to_status,omitempty"`
	HeldBy         *User     `json:"held_by,omitempty"`
}

type ClosureInfo struct {
	RequesterAckResolution bool    `json:"requester_ack_resolution,omitempty"`
	RequesterAckComments   string  `json:"requester_ack_comments,omitempty"`
	ClosureComments        string  `json:"closure_comments,omitempty"`
	ClosureCode            *NameID `json:"closure_code,omitempty"`
}

type CancelFlagComments struct {
	Comment string `json:"comment,omitempty"`
	ID      int64  `json:"id,string,omitempty"`
}

type RequestRequestID struct {
	ID int64 `json:"id,string,omitempty"`
}

type RequestAttachment struct {
	ContentType string `json:"content_type,omitempty"`
	Size        int64  `json:"size,string,omitempty"`
	FileID      int64  `json:"file_id,string,omitempty"`
	Name        string `json:"name,omitempty"`
}

type RequestResources struct {
	Resource string `json:"resource,omitempty"`
}

type RequestResolution struct {
	Content     string `json:"content,omitempty"`
	SubmittedBy *User  `json:"submitted_by,omitempty"`
}

type ServiceApprover struct {
	OrgRoles *NameID `json:"org_roles,omitempty"`
	Users    *User   `json:"service_approvers,omitempty"`
}

// RequestGetListResponse represents the response for fetching a list of requests (tickets).
type RequestGetListResponse struct {
	Entity         string `json:"entity,omitempty"`
	ResponseStatus []struct {
		StatusCode int    `json:"status_code"`
		Status     string `json:"status"`
	} `json:"response_status"`
	ListInfo struct {
		HasMoreRows bool `json:"has_more_rows"`
		StartIndex  int  `json:"start_index"`
		RowCount    int  `json:"row_count"`
	} `json:"list_info"`
	Requests []*RequestAttributes `json:"requests"`
}

// Common subfields for Technician and Requester
// Use this struct for all Technician and Requester fields
type User struct {
	ID         int64  `json:"id,string"`
	Name       string `json:"name"`
	Email      string `json:"email,omitempty"`
	Phone      string `json:"phone,omitempty"`
	Mobile     string `json:"mobile,omitempty"`
	Department *struct {
		ID   int64  `json:"id,string,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"department,omitempty"`
	Site struct {
		ID   int64  `json:"id,string,omitempty"`
		Name string `json:"name,omitempty"`
	} `json:"site,omitempty"`
}

type DateTime struct {
	DisplayValue string `json:"display_value,omitempty"`
	Value        string `json:"value,omitempty"`
}

type NameID struct {
	Name string `json:"name"`
	ID   int64  `json:"id,string,omitempty"`
}
