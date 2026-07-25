package services

import (
	"context"
	"github.com/mieuxvoter/majority-judgment-library-go/judgment"
	"github.com/mieuxvoter/merit-profile-library-go/merit"
	"github.com/sarulabs/di/v2"
	"image/color"
	"log"
	"log/slog"
	"main/src/container"
)

// Analysis is the service that helps analyse the results of a poll
type Analysis struct {
	logger     *slog.Logger
	rasterizer *Rasterizer
}

func (service *Analysis) GenerateMeritProfileSVG(
	proposals []merit.Proposal,
	gradesOutlines [][]uint8,
) (svg string, err error) {

	// Type conversion [][]uint8 → [][]int
	intGradesOutlines := make([][]int, len(gradesOutlines))
	for i, uint8Grades := range gradesOutlines {
		intGradesOutlines[i] = make([]int, len(uint8Grades))
		for j, uintGrade := range uint8Grades {
			intGradesOutlines[i][j] = int(uintGrade)
		}
	}

	svg, err = merit.RenderLinearProfileSVG(
		proposals,
		merit.WithWidth(600),
		merit.WithGradeHeight(48),
		merit.WithBgColor(color.NRGBA{R: 32, G: 32, B: 32, A: 255}),
		// Note: any font used here will need to be available in the Docker image; see Dockerfile
		merit.WithFontFamily("Noto Sans, sans-serif"),
		merit.WithProposalFontSize("28"),
		merit.WithTallyFontSize("20"),
		merit.WithTextOutlineColor(color.White),
		merit.WithTextOutlineWidth(3.0),
		merit.WithBestGradeOnLeft(true),
		merit.WithGradesOutlines(intGradesOutlines),
		merit.WithGradesOutlinesWidth(3.0),
		merit.WithGradesOutlinesColor(color.White),
	)
	if err != nil {
		return
	}

	return
}

func (service *Analysis) GenerateMeritProfilePNG(
	ctx context.Context,
	proposals []merit.Proposal,
	gradesOutlines [][]uint8,
) (png []byte, err error) {

	svg, err := service.GenerateMeritProfileSVG(proposals, gradesOutlines)
	if err != nil {
		return
	}

	return service.rasterizer.ConvertSVGToPNG(ctx, []byte(svg))
}

func (service *Analysis) IsAmountOfJudgmentsEven(
	proposalResult *judgment.ProposalResult,
) bool {
	return proposalResult.Tally.CountJudgments()%2 == 0
}

func (service *Analysis) IsMedianAmbiguous(
	proposalResult *judgment.ProposalResult,
) bool {
	if !service.IsAmountOfJudgmentsEven(proposalResult) {
		return false
	}

	total := proposalResult.Tally.CountJudgments()
	if total == 0 {
		return false
	}

	highMedianIndex := total / 2 // Euclidean division
	lowMedianIndex := highMedianIndex - 1
	highMedianGrade := 0
	lowMedianGrade := 0
	startIndex := uint64(0)
	cursorIndex := uint64(0)
	for gradeIndex, gradeTally := range proposalResult.Tally.Tally {
		if 0 == gradeTally {
			continue
		}

		startIndex = cursorIndex
		cursorIndex += gradeTally
		if (startIndex <= highMedianIndex) && (highMedianIndex < cursorIndex) {
			highMedianGrade = gradeIndex
		}
		if (startIndex <= lowMedianIndex) && (lowMedianIndex < cursorIndex) {
			lowMedianGrade = gradeIndex
		}
	}

	return highMedianGrade != lowMedianGrade
}

func init() {
	err := container.GetBuilder().Add(di.Def{
		Name: "analysis",
		Build: func(ctn di.Container) (interface{}, error) {
			service := &Analysis{
				logger:     ctn.Get("logger").(*slog.Logger),
				rasterizer: ctn.Get("rasterizer").(*Rasterizer),
			}
			return service, nil
		},
	})
	if err != nil {
		log.Fatalln("analysis failed to build", err)
	}
}
