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

// ProblemCreateRequest represents the payload to create a new problem in SDP v3 API.
type ProblemCreateRequest struct {
	Problem struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		ReportedBy  Person `json:"reported_by"`
		Priority    struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"priority,omitempty"`
		Group struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"group,omitempty"`
		Site struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"site,omitempty"`
		Impact struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"impact,omitempty"`
		Urgency struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"urgency,omitempty"`
		Technician  Person `json:"technician,omitempty"`
		CreatedTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"created_time,omitempty"`
		DueByTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"due_by_time,omitempty"`
		// Add other fields as needed
	} `json:"problem"`
}

// ProblemCreateResponse represents the response after creating a problem.
type ProblemCreateResponse struct {
	Problem struct {
		ID          int64  `json:"id,string"`
		Subject     string `json:"subject"`
		Description string `json:"description"`
		ReportedBy  Person `json:"reported_by"`
		Priority    struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"priority,omitempty"`
		Group struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"group,omitempty"`
		Site struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"site,omitempty"`
		Impact struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"impact,omitempty"`
		Urgency struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"urgency,omitempty"`
		Technician  Person `json:"technician,omitempty"`
		CreatedTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"created_time,omitempty"`
		DueByTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"due_by_time,omitempty"`
		// Add other fields as needed
	} `json:"problem"`
}

// ProblemGetResponse represents the response for fetching a problem.
type ProblemGetResponse struct {
	Problem struct {
		ID          int64  `json:"id,string"`
		Subject     string `json:"subject"`
		Description string `json:"description"`
		Status      struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string"`
		} `json:"status"`
		ReportedBy Person `json:"reported_by"`
		Priority   struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"priority,omitempty"`
		Group struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"group,omitempty"`
		Site struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"site,omitempty"`
		Impact struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"impact,omitempty"`
		Urgency struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"urgency,omitempty"`
		Technician  Person `json:"technician,omitempty"`
		CreatedTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"created_time,omitempty"`
		DueByTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"due_by_time,omitempty"`
		// Add other fields as needed
	} `json:"problem"`
}

// TicketCreateRequest represents the payload to create a new ticket in SDP v3 API.
type TicketCreateRequest struct {
	Request struct {
		Subject     string `json:"subject"`
		Description string `json:"description"`
		Requester   Person `json:"requester"`
		Priority    struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"priority,omitempty"`
		Group struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"group,omitempty"`
		Site struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"site,omitempty"`
		Impact struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"impact,omitempty"`
		Urgency struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"urgency,omitempty"`
		Technician  Person `json:"technician,omitempty"`
		CreatedTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"created_time,omitempty"`
		DueByTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"due_by_time,omitempty"`
		// Add other fields as needed
	} `json:"request"`
}

// TicketCreateResponse represents the response after creating a ticket.
type TicketCreateResponse struct {
	Request struct {
		ID          int64  `json:"id,string"`
		Subject     string `json:"subject"`
		Description string `json:"description"`
		Requester   Person `json:"requester"`
		Priority    struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"priority,omitempty"`
		Group struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"group,omitempty"`
		Site struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"site,omitempty"`
		Impact struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"impact,omitempty"`
		Urgency struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"urgency,omitempty"`
		Technician  Person `json:"technician,omitempty"`
		CreatedTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"created_time,omitempty"`
		DueByTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"due_by_time,omitempty"`
		// Add other fields as needed
	} `json:"request"`
}

// TicketGetResponse represents the response for fetching a ticket.
type TicketGetResponse struct {
	Request struct {
		ID          int64  `json:"id,string"`
		Subject     string `json:"subject"`
		Description string `json:"description"`
		Status      struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string"`
		} `json:"status"`
		Requester Person `json:"requester"`
		Priority  struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"priority,omitempty"`
		Group struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"group,omitempty"`
		Site struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"site,omitempty"`
		Impact struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"impact,omitempty"`
		Urgency struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"urgency,omitempty"`
		Technician  Person `json:"technician,omitempty"`
		CreatedTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"created_time,omitempty"`
		DueByTime struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"due_by_time,omitempty"`
		// Add other fields as needed
	} `json:"request"`
}

// RequestGetListResponse represents the response for fetching a list of requests (tickets).
type RequestGetListResponse struct {
	Requests []struct {
		ID          int64  `json:"id,string"`
		Subject     string `json:"subject"`
		Description string `json:"description,omitempty"`
		Status      *struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string"`
		} `json:"status"`
		Requester *Person `json:"requester"`
		Priority  *struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"priority,omitempty"`
		Group *struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"group,omitempty"`
		Site *struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"site,omitempty"`
		Impact *struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"impact,omitempty"`
		Urgency *struct {
			Name string `json:"name"`
			ID   int64  `json:"id,string,omitempty"`
		} `json:"urgency,omitempty"`
		Technician  *Person `json:"technician"`
		CreatedTime *struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"created_time,omitempty"`
		DueByTime *struct {
			DisplayValue string `json:"display_value,omitempty"`
			Value        string `json:"value,omitempty"`
		} `json:"due_by_time,omitempty"`
		// Add other fields as needed
	} `json:"requests"`
	// Pagination and other metadata fields can be added if needed
}

// Common subfields for Technician and Requester
// Use this struct for all Technician and Requester fields
type Person struct {
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
