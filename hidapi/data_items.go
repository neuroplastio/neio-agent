package hidapi

import "github.com/neuroplastio/neio-agent/hidapi/hiddesc"

type DataItemSet struct {
	reportIDs []uint8
	dataItems map[uint8][]hiddesc.DataItem
	types     map[uint8]map[int]hiddesc.MainItemType
}

type DataItemSetReport struct {
	ID        uint8
	DataItems []hiddesc.DataItem
}

func NewDataItemSet(desc hiddesc.ReportDescriptor) *DataItemSet {
	set := &DataItemSet{
		dataItems: make(map[uint8][]hiddesc.DataItem),
		types:     make(map[uint8]map[int]hiddesc.MainItemType),
	}
	var drill func(c hiddesc.Collection)
	drill = func(c hiddesc.Collection) {
		for _, item := range c.Items {
			if item.Collection == nil {
				set.Add(item.Type, *item.DataItem)
			} else {
				drill(*item.Collection)
			}
		}
	}
	for _, collection := range desc.Collections {
		drill(collection)
	}
	return set
}

func (s *DataItemSet) Report(reportID uint8) []hiddesc.DataItem {
	return s.dataItems[reportID]
}

func (s *DataItemSet) Reports() []DataItemSetReport {
	reports := make([]DataItemSetReport, 0, len(s.reportIDs))
	for _, reportID := range s.reportIDs {
		reports = append(reports, DataItemSetReport{
			ID:        reportID,
			DataItems: s.dataItems[reportID],
		})
	}
	return reports
}

func (s *DataItemSet) WithType(typ hiddesc.MainItemType) DataItemSet {
	set := DataItemSet{
		dataItems: make(map[uint8][]hiddesc.DataItem),
		types:     make(map[uint8]map[int]hiddesc.MainItemType),
	}
	for _, report := range s.Reports() {
		for idx, item := range report.DataItems {
			if s.types[report.ID][idx] == typ {
				set.Add(typ, item)
			}
		}
	}
	return set
}

func (s *DataItemSet) Add(typ hiddesc.MainItemType, item hiddesc.DataItem) {
	_, ok := s.dataItems[item.ReportID]
	if !ok {
		s.reportIDs = append(s.reportIDs, item.ReportID)
		s.types[item.ReportID] = make(map[int]hiddesc.MainItemType)
	}
	s.dataItems[item.ReportID] = append(s.dataItems[item.ReportID], item)
	s.types[item.ReportID][len(s.dataItems[item.ReportID])-1] = typ
}

func (s *DataItemSet) Type(reportID uint8, idx int) hiddesc.MainItemType {
	return s.types[reportID][idx]
}

func (s *DataItemSet) HasReportID() bool {
	_, hasReportIDZero := s.dataItems[0]
	return !hasReportIDZero
}

func (s *DataItemSet) MakeDescriptor() hiddesc.ReportDescriptor {
	desc := hiddesc.ReportDescriptor{
		Collections: make([]hiddesc.Collection, 0, len(s.reportIDs)),
	}
	for _, reportID := range s.reportIDs {
		items := s.dataItems[reportID]
		collection := hiddesc.Collection{
			Items: make([]hiddesc.MainItem, 0, len(items)),
		}
		for idx, item := range items {
			collection.Items = append(collection.Items, hiddesc.MainItem{
				Type:     s.types[reportID][idx],
				DataItem: &item,
			})
		}
		desc.Collections = append(desc.Collections, collection)
	}
	return desc
}
