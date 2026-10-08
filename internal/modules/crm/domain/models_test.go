package domain

import "testing"

func TestTerminalStageSemantics(t *testing.T) {
	if err := (Stage{Key: "won", Name: "Won", Category: StageWon, Probability: 90}).Validate(); err == nil {
		t.Fatal("won probability must be fixed")
	}
	stages := []Stage{{Key: "new", Name: "New", Category: StageOpen, Position: 0, Active: true}, {Key: "won", Name: "Won", Category: StageWon, Position: 1, Probability: 100, Active: true}, {Key: "lost", Name: "Lost", Category: StageLost, Position: 2, Probability: 0, Active: true}}
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
		stages = append(stages, Stage{Key: "won", Name: "Won", Category: StageWon, Position: len(stages), Probability: 100, Active: true}, Stage{Key: "lost", Name: "Lost", Category: StageLost, Position: len(stages) + 1, Probability: 0, Active: true})
		if err := ValidatePipelineStages(stages); err != nil {
			t.Fatalf("%s: %v", template.Key, err)
		}
	}
}
