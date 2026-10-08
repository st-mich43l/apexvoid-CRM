package domain

import "testing"

func TestTerminalStageSemantics(t *testing.T) {
	if err := (Stage{Key: "won", Name: "Won", Category: StageWon, Probability: 90}).Validate(); err == nil {
		t.Fatal("won probability must be fixed")
	}
	stages := []Stage{{Key: "new", Name: "New", Category: StageOpen, Active: true}, {Key: "won", Name: "Won", Category: StageWon, Probability: 100, Active: true}, {Key: "lost", Name: "Lost", Category: StageLost, Active: true}}
	if err := ValidatePipelineStages(stages); err != nil {
		t.Fatal(err)
	}
}
func TestTemplatesHaveValidStageConfiguration(t *testing.T) {
	for _, template := range Templates() {
		stages := []Stage{}
		for index, item := range template.OpenStages {
			stages = append(stages, Stage{Key: item.Key, Name: item.Name, Category: StageOpen, Position: index, Probability: item.Probability, Active: true})
		}
		stages = append(stages, Stage{Key: "won", Name: "Won", Category: StageWon, Probability: 100, Active: true}, Stage{Key: "lost", Name: "Lost", Category: StageLost, Probability: 0, Active: true})
		if err := ValidatePipelineStages(stages); err != nil {
			t.Fatalf("%s: %v", template.Key, err)
		}
	}
}
