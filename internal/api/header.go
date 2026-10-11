package api

import (
	"net/http"
	"time"
)

const (
	headerTimeFormat = "Mon, _2 Jan 2006 15:04:05 MST"
)

const (
	HeaderBNNamespace      = "Battlenet-Namespace"
	HeaderBNSchema         = "Battlenet-Schema"
	HeaderBNSchemaRevision = "Battlenet-Schema-Revision"
	HeaderContentType      = "Content-Type"
	HeaderDate             = "Date"
	HeaderLastModified     = "Last-Modified"
)

type Header struct {
	BNNamespace      string    `json:"Battlenet-Namespace"`
	BNSchema         string    `json:"Battlenet-Schema"`
	BNSchemaRevision string    `json:"Battlenet-Schema-Revision"`
	ContentType      string    `json:"Content-Type"`
	Date             time.Time `json:"Date"`
	LastModified     time.Time `json:"Last-Modified"`
}

func ParseHeader(httpHeader http.Header) *Header {
	header := Header{
		BNNamespace:      httpHeader.Get(HeaderBNNamespace),
		BNSchema:         httpHeader.Get(HeaderBNSchema),
		BNSchemaRevision: httpHeader.Get(HeaderBNSchemaRevision),
		ContentType:      httpHeader.Get(HeaderContentType),
	}
	if httpHeader.Get(HeaderDate) != "" {
		var err error
		header.Date, err = time.Parse(headerTimeFormat, httpHeader.Get(HeaderDate))
		if err != nil {
			header.Date = time.Time{}
		}
	}
	if httpHeader.Get(HeaderLastModified) != "" {
		var err error
		header.LastModified, err = time.Parse(headerTimeFormat, httpHeader.Get(HeaderLastModified))
		if err != nil {
			header.LastModified = time.Time{}
		}
	}

	return &header
}
