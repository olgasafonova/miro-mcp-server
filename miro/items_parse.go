package miro

import "encoding/json"

// =============================================================================
// Item Wire Format Parsing (raw JSON to ItemSummary / GetItemResult)
// =============================================================================

// rawItemPosition mirrors the JSON wire format for an item's position.
type rawItemPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// rawItemGeometry mirrors the JSON wire format for an item's geometry block.
type rawItemGeometry struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// rawItemStyle mirrors the JSON wire format for an item's style block.
type rawItemStyle struct {
	FillColor   string `json:"fillColor"`
	TextAlign   string `json:"textAlign"`
	BorderColor string `json:"borderColor"`
	FontSize    string `json:"fontSize"`
}

// rawItemUser mirrors the JSON wire format for createdBy / modifiedBy actors.
type rawItemUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// rawItemSummary mirrors the JSON wire format used by parseItemSummary.
// Extended fields (Geometry, Style, timestamps, actors) are only populated
// when ListItems is called with detail_level=full.
type rawItemSummary struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Position *rawItemPosition `json:"position"`
	ParentID string           `json:"parentId"`
	Data     struct {
		Content string `json:"content"`
		Title   string `json:"title"`
		Shape   string `json:"shape"`
	} `json:"data"`
	Geometry   *rawItemGeometry `json:"geometry"`
	Style      *rawItemStyle    `json:"style"`
	CreatedAt  string           `json:"createdAt"`
	ModifiedAt string           `json:"modifiedAt"`
	CreatedBy  *rawItemUser     `json:"createdBy"`
	ModifiedBy *rawItemUser     `json:"modifiedBy"`
}

// parseItemSummary extracts an ItemSummary from raw JSON data.
// When fullDetails is true, additional fields are populated.
func parseItemSummary(raw json.RawMessage, fullDetails bool) ItemSummary {
	var base rawItemSummary
	if err := json.Unmarshal(raw, &base); err != nil {
		return ItemSummary{}
	}
	item := minimalItemSummary(base)
	if fullDetails {
		addItemFullDetails(&item, base)
	}
	return item
}

// minimalItemSummary builds the minimum-detail ItemSummary returned for
// every list response (regardless of detail_level).
func minimalItemSummary(base rawItemSummary) ItemSummary {
	content := base.Data.Content
	if content == "" {
		content = base.Data.Title
	}
	item := ItemSummary{
		ID:       base.ID,
		Type:     base.Type,
		Content:  content,
		ParentID: base.ParentID,
	}
	if base.Position != nil {
		item.X = base.Position.X
		item.Y = base.Position.Y
	}
	return item
}

// addItemFullDetails populates the extended fields when detail_level=full.
func addItemFullDetails(item *ItemSummary, base rawItemSummary) {
	if base.Geometry != nil {
		item.Width = base.Geometry.Width
		item.Height = base.Geometry.Height
	}
	if base.Style != nil {
		item.Style = &ItemStyleInfo{
			FillColor:   base.Style.FillColor,
			TextAlign:   base.Style.TextAlign,
			BorderColor: base.Style.BorderColor,
			FontSize:    base.Style.FontSize,
		}
	}
	// The shape kind arrives in data.shape, not style; fold it into the
	// style info so consumers see one styling surface.
	if base.Data.Shape != "" {
		if item.Style == nil {
			item.Style = &ItemStyleInfo{}
		}
		item.Style.Shape = base.Data.Shape
	}
	item.CreatedAt = base.CreatedAt
	item.ModifiedAt = base.ModifiedAt
	if base.CreatedBy != nil {
		item.CreatedBy = &UserInfo{ID: base.CreatedBy.ID, Name: base.CreatedBy.Name}
	}
	if base.ModifiedBy != nil {
		item.ModifiedBy = &UserInfo{ID: base.ModifiedBy.ID, Name: base.ModifiedBy.Name}
	}
}

// rawItemDetail mirrors the JSON wire format used by GetItem.
type rawItemDetail struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Position *rawItemPosition `json:"position"`
	Geometry *rawItemGeometry `json:"geometry"`
	Data     struct {
		Content string `json:"content"`
		Title   string `json:"title"`
		Shape   string `json:"shape"`
	} `json:"data"`
	Style struct {
		FillColor string `json:"fillColor"`
	} `json:"style"`
	ParentID   string `json:"parentId"`
	CreatedAt  string `json:"createdAt"`
	ModifiedAt string `json:"modifiedAt"`
	CreatedBy  *struct {
		Name string `json:"name"`
	} `json:"createdBy"`
	ModifiedBy *struct {
		Name string `json:"name"`
	} `json:"modifiedBy"`
}

// buildGetItemResult assembles the GetItemResult from a raw item record,
// folding the four optional-pointer-deref blocks into a flat assignment chain.
func buildGetItemResult(item rawItemDetail) GetItemResult {
	result := GetItemResult{
		ID:         item.ID,
		Type:       item.Type,
		Content:    item.Data.Content,
		Title:      item.Data.Title,
		Shape:      item.Data.Shape,
		Color:      item.Style.FillColor,
		ParentID:   item.ParentID,
		CreatedAt:  item.CreatedAt,
		ModifiedAt: item.ModifiedAt,
	}
	if item.Position != nil {
		result.X = item.Position.X
		result.Y = item.Position.Y
	}
	if item.Geometry != nil {
		result.Width = item.Geometry.Width
		result.Height = item.Geometry.Height
	}
	if item.CreatedBy != nil {
		result.CreatedBy = item.CreatedBy.Name
	}
	if item.ModifiedBy != nil {
		result.ModifiedBy = item.ModifiedBy.Name
	}
	return result
}
