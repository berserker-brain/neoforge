package neoforge

import (
	"fmt"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type Stats struct {
	StatementType        string
	ResultAvailableAfter time.Duration
	ResultConsumedAfter  time.Duration
	Notifications        []neo4j.Notification
	//Counters
	NodesCreated         int
	NodesDeleted         int
	RelationshipsCreated int
	RelationshipDeleted  int
	PropertiesSet        int
	LabelsAdded          int
	LabelsRemoved        int
	IndexesAdded         int
	IndexesRemoved       int
	ConstraintsAdded     int
	ConstraintsRemoved   int
	SystemUpdates        int
}

func (s *Stats) FromResultSummary(summary neo4j.ResultSummary) {
	s.StatementType = summary.StatementType().String()
	s.ResultAvailableAfter = summary.ResultAvailableAfter()
	s.ResultConsumedAfter = summary.ResultConsumedAfter()
	s.Notifications = summary.Notifications()

	sum := summary.Counters()
	if sum.ContainsSystemUpdates() {
		s.SystemUpdates = sum.SystemUpdates()
	}

	if sum.ContainsUpdates() {
		s.NodesCreated = sum.NodesCreated()
		s.NodesDeleted = sum.NodesDeleted()
		s.RelationshipsCreated = sum.RelationshipsCreated()
		s.RelationshipDeleted = sum.RelationshipsDeleted()
		s.PropertiesSet = sum.PropertiesSet()
		s.LabelsAdded = sum.LabelsAdded()
		s.LabelsRemoved = sum.LabelsRemoved()
		s.IndexesAdded = sum.IndexesAdded()
		s.IndexesRemoved = sum.IndexesRemoved()
		s.ConstraintsAdded = sum.ConstraintsAdded()
		s.ConstraintsRemoved = sum.ConstraintsRemoved()
	}

	s.PrintNotifications(true)
}

func (s *Stats) Print() {
	fmt.Println("Result Available After: ", s.ResultAvailableAfter)
	fmt.Println("Result Consumed After: ", s.ResultConsumedAfter)

	if s.NodesCreated != 0 {
		fmt.Printf("Nodes Created: %d\n", s.NodesCreated)
	}
	if s.NodesDeleted != 0 {
		fmt.Printf("Nodes Deleted: %d\n", s.NodesDeleted)
	}
	if s.RelationshipsCreated != 0 {
		fmt.Printf("Relationships Created: %d\n", s.RelationshipsCreated)
	}
	if s.RelationshipDeleted != 0 {
		fmt.Printf("Relationship Deleted: %d\n", s.RelationshipDeleted)
	}
	if s.PropertiesSet != 0 {
		fmt.Printf("Properties Set: %d\n", s.PropertiesSet)
	}
	if s.LabelsAdded != 0 {
		fmt.Printf("Labels Added: %d\n", s.LabelsAdded)
	}
	if s.LabelsRemoved != 0 {
		fmt.Printf("Labels Removed: %d\n", s.LabelsRemoved)
	}
	if s.IndexesAdded != 0 {
		fmt.Printf("Indexes Added: %d\n", s.IndexesAdded)
	}
	if s.IndexesRemoved != 0 {
		fmt.Printf("Indexes Removed: %d\n", s.IndexesRemoved)
	}
	if s.ConstraintsAdded != 0 {
		fmt.Printf("Constraints Added: %d\n", s.ConstraintsAdded)
	}
	if s.ConstraintsRemoved != 0 {
		fmt.Printf("Constraints Removed: %d\n", s.ConstraintsRemoved)
	}
	if s.SystemUpdates != 0 {
		fmt.Printf("System Updates: %d\n", s.SystemUpdates)
	}

	s.PrintNotifications(false)
}

func (s *Stats) PrintNotifications(onlyPrintWarnings bool) {
	for _, notification := range s.Notifications {
		if onlyPrintWarnings &&
			(notification.RawSeverityLevel() != "WARNING" ||
				strings.Contains(notification.Code(), "UnknownLabelWarning")) {
			continue
		}
	}
}
