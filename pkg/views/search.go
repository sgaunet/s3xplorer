package views

import (
	"context"
	"io"

	"github.com/sgaunet/s3xplorer/pkg/dto"
)

// SearchData is the data structure used to render the search page
type SearchData struct {
	ActualFolder string
	Objects      []dto.S3Object
	SearchStr    string
}

// RenderSearch renders the search page
func (v *Views) RenderSearch(w io.Writer, data SearchData) {
	RenderSearch(data.SearchStr, data.ActualFolder, data.Objects).Render(context.TODO(), w)
}
