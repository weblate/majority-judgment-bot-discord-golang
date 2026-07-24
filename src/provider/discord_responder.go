package provider

import (
	"bytes"
	"context"
	"fmt"
	"github.com/disgoorg/disgo/discord"
	"github.com/mieuxvoter/majority-judgment-library-go/judgment"
	"github.com/mieuxvoter/merit-profile-library-go/merit"
	"github.com/mozillazg/go-slugify"
	"github.com/sarulabs/di/v2"
	"log"
	"log/slog"
	"main/src/container"
	db "main/src/database"
	"main/src/locales"
	"main/src/security"
	"main/src/services"
	"strings"
	"time"
	"xorm.io/xorm"
)

// DiscordResponder implements provider.ResponderInterface for Discord
type DiscordResponder struct {
	orm          *xorm.Engine
	analysis     *services.Analysis
	localization *locales.Localization
	logger       *slog.Logger
}

func (r DiscordResponder) sanitizeTitle(title string) string {
	return security.TruncateString(title, 256)
}

func (r DiscordResponder) Matches(input Input) bool {
	_, isDiscord := (input).(DiscordCommandInput)
	if !isDiscord {
		_, isDiscord = (input).(DiscordButtonInput)
	}
	return isDiscord
}

func (r DiscordResponder) RespondMessage(
	input Input,
	message string,
	ephemeral bool,
) error {
	if d, isDiscord := (input).(DiscordInteraction); isDiscord {
		msg := discord.MessageCreate{
			Content: message,
		}

		if ephemeral {
			msg = msg.WithFlags(discord.MessageFlagEphemeral)
		}

		return d.CreateMessage(msg)
	}

	return RaiseInvalidProviderError("Discord:RespondMessage")
}

func (r DiscordResponder) RespondPollView(
	input Input,
	poll *db.Poll,
	proposals []*db.Proposal,
	replaceMessage bool,
) error {
	if d, isDiscord := input.(DiscordCommandInput); isDiscord {

		localizer := r.localization.GetLocalizer(input.GetActorLanguage())

		title := "### :scales: " + r.sanitizeTitle(poll.Subject)
		//amountOfVotes, _ := db.CountBallots(r.orm, poll) // always zero at this point

		description := ""
		if len(proposals) > 0 {
			for _, proposal := range proposals {
				description += "- "
				proposalName := security.RemoveMarkdown(proposal.Name)
				description += security.TruncateEllipsis(proposalName, 256)
				description += "\n"
			}
		}

		msg := discord.MessageCreate{
			Flags: discord.MessageFlagIsComponentsV2,
			Components: []discord.LayoutComponent{
				discord.NewContainer(
					discord.NewSection(
						discord.NewTextDisplay(title),
						discord.NewTextDisplay(description),
					).WithAccessory(
						discord.NewPrimaryButton(
							localizer.T("ActionVote"),
							fmt.Sprintf("/button/poll/%d/vote", poll.Id),
						).WithEmoji(
							discord.ComponentEmoji{Name: "🗳"},
						),
					),
					discord.NewSmallSeparator(),
					discord.NewSection(
						discord.NewTextDisplay(
							fmt.Sprintf(":closed_lock_with_key: ")+localizer.T("SecretBallot"),
							//fmt.Sprintf(":eyes:", amountOfVotes),
						),
					).WithAccessory(
						discord.NewSecondaryButton(
							localizer.T("ActionShowResults"),
							fmt.Sprintf("/button/poll/%d/results", poll.Id),
						).WithEmoji(
							discord.ComponentEmoji{Name: "🔎"},
							//discord.ComponentEmoji{Name: "🏆"},
						),
					),
				),
			},
		}

		return d.Event.CreateMessage(msg)
	}

	return RaiseInvalidProviderError("Discord:RespondPollView")
}

func (r DiscordResponder) RespondJudgmentUi(
	input Input,
	proposal *db.Proposal,
	poll *db.Poll,
	previousJudgment *db.Judgment,
	replaceMessage bool,
) error {
	if d, isDiscord := input.(DiscordInteraction); isDiscord {

		title := "### " + security.TruncateString(proposal.Name, 256)

		gradeButtons := make([]discord.InteractiveComponent, 0, 5)
		for gradeLevel, grade := range poll.GetGradingSlice(services.GetGradings()) {
			customId := fmt.Sprintf("/button/poll/%d/judge/%d/as/%d", poll.Id, proposal.Id, gradeLevel)
			emoji := discord.ComponentEmoji{Name: grade}
			button := discord.NewSecondaryButton("", customId).WithEmoji(emoji)
			if previousJudgment != nil && previousJudgment.Grade == uint8(gradeLevel) {
				button = discord.NewSuccessButton("", customId).WithEmoji(emoji)
			}
			gradeButtons = append(gradeButtons, button)
		}

		flags := discord.MessageFlagIsComponentsV2 | discord.MessageFlagEphemeral
		components := []discord.LayoutComponent{
			discord.NewContainer(
				discord.NewTextDisplay(title),
				discord.NewActionRow(gradeButtons...),
			),
		}

		if replaceMessage {
			return d.UpdateMessage(discord.MessageUpdate{
				// We need pointers here, not sure why
				Flags:      &flags,
				Components: &components,
			})
		}

		return d.CreateMessage(discord.MessageCreate{
			Flags:      flags,
			Components: components,
		})
	}

	return RaiseInvalidProviderError("Discord:RespondJudgmentUi")
}

func (r DiscordResponder) RespondBallotSummary(
	input Input,
	poll *db.Poll,
	proposals []db.Proposal,
	judgments []db.Judgment,
) error {
	if d, isDiscord := input.(DiscordInteraction); isDiscord {

		localizer := r.localization.GetLocalizer(input.GetActorLanguage())

		title := fmt.Sprintf("### ✅ **%s**", localizer.T("BallotSummaryVoteRecorded"))
		message := localizer.T("BallotSummaryHereIsSummary")
		summary := ""
		for k := range judgments {
			grade := poll.GetGradeIcon(services.GetGradings(), judgments[k].Grade)
			summary += fmt.Sprintf("- %s ⋅ %s\n", grade, proposals[k].Name)
		}

		flags := discord.MessageFlagIsComponentsV2 | discord.MessageFlagEphemeral
		components := []discord.LayoutComponent{
			discord.NewContainer(
				discord.NewTextDisplay(title),
				discord.NewTextDisplay(message),
				discord.NewTextDisplay(summary),
			),
		}

		return d.UpdateMessage(discord.MessageUpdate{
			Flags:      &flags,
			Components: &components,
		})
	}

	return RaiseInvalidProviderError("Discord:RespondBallotSummary")
}

func (r DiscordResponder) RespondPollResult(
	input Input,
	poll *db.Poll,
	proposals []db.Proposal,
	pollTally *judgment.PollTally,
	pollResult *judgment.PollResult,
	title string,
	message string,
	asPrivateMessage bool,
	canInspect bool,
) error {
	if d, isDiscord := input.(DiscordInteraction); isDiscord {

		localizer := r.localization.GetLocalizer(input.GetActorLanguage())

		// Our PNG generation is super long (~5s), so let's use a deferred response
		deferErr := d.DeferCreateMessage(asPrivateMessage)
		if deferErr != nil {
			return deferErr
		}

		rendererProposals := make([]merit.Proposal, len(proposals))
		gradesOutlines := make([][]uint8, len(proposals))
		for i, proposal := range pollResult.ProposalsSorted {
			gradesOutlines[i] = []uint8{proposal.Analysis.MedianGrade}
			rendererProposals[i] = merit.Proposal{
				Name:  proposals[proposal.Index].Name,
				Tally: pollTally.Proposals[proposal.Index].Tally,
			}
		}

		isAnyMedianAmbiguous := false
		for _, proposalResult := range pollResult.ProposalsSorted {
			if r.analysis.IsMedianAmbiguous(proposalResult) {
				isAnyMedianAmbiguous = true
				break
			}
		}

		ctx, cancel := context.WithTimeout(context.TODO(), 120*time.Second)
		defer cancel()

		// Discord does not render SVG files (although it's somewhat safe in img tags, right?)
		// So we need to create a raster version of our merit profile.
		pngBytes, err := r.analysis.GenerateMeritProfilePNG(
			ctx,
			rendererProposals,
			gradesOutlines,
		)
		if err != nil {
			return err
		}

		pollSlug := strings.Trim(
			security.TruncateString(
				slugify.Slugify(poll.Subject),
				128,
			),
			"-",
		)
		if len(pollSlug) == 0 {
			pollSlug = "unnamed"
		}
		imageFilenameNoExt := fmt.Sprintf(
			`merit-profile-%s-%s`,
			time.Now().Format("20060102"),
			pollSlug,
		)
		imageDescription := localizer.Tf(
			"DescriptionOfMeritProfileOfPoll",
			map[string]interface{}{
				"Name": poll.Subject,
			},
		)

		footerLeft := discord.NewTextDisplay(
			localizer.Tfp(
				"SomeParticipants",
				map[string]interface{}{
					"Amount": int(pollTally.AmountOfJudges),
				},
				int(pollTally.AmountOfJudges),
			),
		)

		flags := discord.MessageFlagIsComponentsV2
		if asPrivateMessage {
			flags |= discord.MessageFlagEphemeral
		}

		msgContainer := discord.NewContainer(
			discord.NewTextDisplay("### "+title),
			discord.NewTextDisplay(message),
			discord.NewSmallSeparator(),
			discord.NewMediaGallery(
				discord.MediaGalleryItem{
					Media: discord.UnfurledMediaItem{
						URL: "attachment://" + imageFilenameNoExt + ".png",
					},
					Description: imageDescription,
					Spoiler:     false,
				},
			),
			// Shows a link to download the file. (not what we want)
			//discord.NewFileComponent(
			//	"attachment://"+imageFilenameNoExt+".png",
			//),
		)

		if isAnyMedianAmbiguous {
			msgContainer = msgContainer.AddComponents(
				discord.NewTextDisplay("> 🔬 " + localizer.T("InformAboutAmbiguousMedian")),
			)
		}

		if asPrivateMessage {
			msgContainer = msgContainer.AddComponents(
				discord.NewSmallSeparator(),
				discord.NewSection(
					footerLeft,
				).WithAccessory(
					discord.NewSecondaryButton(
						localizer.T("ActionPublish"),
						fmt.Sprintf("/button/poll/%d/publish", poll.Id),
					).WithEmoji(
						discord.ComponentEmoji{Name: "📢"},
					),
				),
			)
		} else {
			msgContainer = msgContainer.AddComponents(
				discord.NewSmallSeparator(),
				footerLeft,
			)
		}

		msg := discord.MessageUpdate{
			Flags: &flags,
			Components: &[]discord.LayoutComponent{
				msgContainer,
			},
			Files: []*discord.File{
				discord.NewFile(
					imageFilenameNoExt+".png",
					imageDescription,
					bytes.NewReader(pngBytes),
					discord.FileFlagsNone,
				),
				//discord.NewFile(
				//	imageFilenameNoExt+".svg",
				//	imageDescription,
				//	bytes.NewReader(svgBytes),
				//	discord.FileFlagsNone,
				//),
			},
		}

		return d.UpdateDeferredMessage(msg)
	}

	return RaiseInvalidProviderError("Discord:RespondPollResult")
}

//func (r DiscordResponder) RespondBallotsInspection(
//	input provider.Input,
//	poll *db.Poll,
//	proposals []db.Proposal,
//	judgments []db.Judgment,
//) error {
//
//	csvString := "judge_id"
//	for _, proposal := range proposals {
//		csvString += fmt.Sprintf(", \"%s\"", security.EscapeCsvValue(proposal.Name))
//	}
//
//	currentJudgeVendorId := ""
//	for k, jt := range judgments {
//		if jt.JudgeSnowflake == currentJudgeVendorId {
//			continue
//		}
//		currentJudgeVendorId = jt.JudgeSnowflake
//		csvString += fmt.Sprintf("\n\"%s\"", currentJudgeVendorId)
//
//		// Extra complexity is to handle missing judgments
//		pk := 0
//		for _, proposal := range proposals {
//			judgmentOfJudge := judgments[k+pk]
//			val := "0"
//			if judgmentOfJudge.ProposalId == proposal.Id {
//				val = fmt.Sprint(judgmentOfJudge.Grade)
//				pk += 1
//			}
//			csvString += ", " + security.EscapeCsvValue(val)
//		}
//	}
//
//	if d, isDiscord := input.(provider.DiscordCommandInput); isDiscord {
//
//		messageType := disgord.InteractionCallbackChannelMessageWithSource
//		csvFile := disgord.CreateMessageFile{
//			Reader:   strings.NewReader(csvString),
//			FileName: fmt.Sprintf("poll_%d.csv", poll.Id),
//		}
//		content := fmt.Sprintf("🏛 Here are the individual ballots for the poll **%s** :", poll.Subject)
//		interactionResponse := &disgord.CreateInteractionResponse{
//			Type: messageType,
//			Data: &disgord.CreateInteractionResponseData{
//				Flags: disgord.MessageFlagEphemeral,
//				Files: []disgord.CreateMessageFile{
//					csvFile,
//				},
//				Content: content,
//			},
//		}
//
//		return d.Session.SendInteractionResponse(d.Context, d.Interaction, interactionResponse)
//	}
//
//	return provider.RaiseInvalidProviderError("Discord:RespondBallotInspection")
//}

func (r DiscordResponder) RespondServerError(
	input Input,
	message string,
) error {
	if _, isDiscord := input.(DiscordInteraction); isDiscord {
		return r.RespondMessage(
			input,
			fmt.Sprintf(
				"### 💥 **BOOM !**\n"+
					"\n"+
					"%s\n",
				message,
			),
			true,
		)
	}

	return RaiseInvalidProviderError("Discord:RespondServerError")
}

func (r DiscordResponder) RespondUserError(
	input Input,
	message string,
) error {
	if _, isDiscord := input.(DiscordInteraction); isDiscord {
		return r.RespondMessage(
			input,
			fmt.Sprintf(
				"### 🤖🗯\n"+
					"\n"+
					"%s\n"+
					"",
				message,
			),
			true,
		)
	}

	return RaiseInvalidProviderError("Discord:RespondUserError")
}

func init() {
	err := container.GetBuilder().Add(di.Def{
		Name: "responder.discord",
		Build: func(ctn di.Container) (interface{}, error) {
			responder := &DiscordResponder{
				orm:          ctn.Get("database.engine").(*xorm.Engine),
				analysis:     ctn.Get("analysis").(*services.Analysis),
				localization: ctn.Get("localization").(*locales.Localization),
				logger:       ctn.Get("logger").(*slog.Logger),
			}
			return responder, nil
		},
	})

	if err != nil {
		log.Fatalln("service responder.discord failed to build", err)
	}
}
