package mmf

import (
	"fmt"
	"time"

	"agones.dev/agones/pkg/util/runtime"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

var logger = runtime.NewLoggerWithSource("mmf")

// This match function fetches all the Tickets for all the pools specified in
// the profile. It uses a configured number of tickets from each pool to generate
// a Match Proposal. It continues to generate proposals till one of the pools
// runs out of Tickets.
const (
	matchName              = "pixo-matchfunction"
	ticketsPerPoolPerMatch = 1
)

// Run is this match function's implementation of the gRPC call defined in api/matchfunction.proto.
func (s *MatchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	// Fetch tickets for the pools specified in the Match Profile.
	logger.Debugf("Generating proposals for function %s", req.GetProfile().GetName())

	poolTickets, err := matchfunction.QueryPools(stream.Context(), s.queryServiceClient, req.GetProfile().GetPools())
	if err != nil {
		logger.WithError(err).Error("Failed to query tickets for the given pools")
		return err
	}
	// logger.Info(poolTickets)

	// Generate proposals.
	proposals, err := makeMatches(req.GetProfile(), poolTickets)
	if err != nil {
		logger.Errorf("Failed to generate matches, got %s", err.Error())
		return err
	}

	logger.Debugf("Streaming %v proposals to Open Match", len(proposals))
	// Stream the generated proposals back to Open Match.
	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			logger.Errorf("Failed to stream proposals to Open Match, got %s", err.Error())
			return err
		}
	}

	return nil
}

func makeMatches(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	// tickets, ok := poolTickets["everyone"]// filter by only the  name everyone -> from director
	// if !ok {
	// 	return nil, errors.New("Expected pool named everyone.") // this to be changed
	// }

	var matches []*pb.Match
	count := 0
	for {
		insufficientTickets := false
		matchTickets := []*pb.Ticket{}
		for pool, tickets := range poolTickets {
			if len(tickets) < ticketsPerPoolPerMatch {
				// This pool is completely drained out. Stop creating matches.
				insufficientTickets = true
				break
			}
			// Remove the Tickets from this pool and add to the match proposal.
			matchTickets = append(matchTickets, tickets[0:ticketsPerPoolPerMatch]...)
			poolTickets[pool] = tickets[ticketsPerPoolPerMatch:]
		}

		if insufficientTickets {
			logger.Debug("There is no sufficient tickets to pair")
			break
		}

		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), count),
			MatchProfile:  p.GetName(),
			MatchFunction: matchName,
			Tickets:       matchTickets,
		})

		count++
	}

	return matches, nil
}
