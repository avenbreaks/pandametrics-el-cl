package state

import (
	"sync"

	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db"
	"github.com/cortze/eth-cl-state-analyzer/pkg/db/model"
	"github.com/cortze/eth-cl-state-analyzer/pkg/state_metrics"
	"github.com/cortze/eth-cl-state-analyzer/pkg/utils"
)

func (s *StateAnalyzer) runProcessState(wgProcess *sync.WaitGroup, downloadFinishedFlag *bool) {
	defer wgProcess.Done()
	log.Info("Launching Beacon State Pre-Processer")
loop:
	for {
		// in case the downloads have finished, and there are no more tasks to execute
		if *downloadFinishedFlag && len(s.EpochTaskChan) == 0 {
			log.Warn("the task channel has been closed, finishing epoch routine")

			break loop
		}

		select {
		case <-s.ctx.Done():
			log.Info("context has died, closing state processer routine")
			return

		case task, ok := <-s.EpochTaskChan:

			// check if the channel has been closed
			if !ok {
				log.Warn("the task channel has been closed, finishing epoch routine")
				return
			}
			log.Infof("epoch task received for slot %d, epoch: %d, analyzing...", task.State.Slot, task.State.Epoch)

			// returns the state in a custom struct for Phase0, Altair of Bellatrix
			stateMetrics, err := state_metrics.StateMetricsByForkVersion(task.NextState, task.State, task.PrevState, s.cli.Api)

			if err != nil {
				log.Errorf(err.Error())
				continue
			}
			if task.NextState.Slot <= s.FinalSlot || task.Finalized {
				log.Debugf("Creating validator batches for slot %d...", task.State.Slot)
				// divide number of validators into number of workers equally

				var validatorBatches []utils.PoolKeys

				// first of all see if there user input any validator list
				// in case no validators provided, do all the existing ones in the next epoch
				valIdxs := stateMetrics.GetMetricsBase().NextState.GetAllVals()
				validatorBatches = utils.DivideValidatorsBatches(valIdxs, s.validatorWorkerNum)

				if len(s.PoolValidators) > 0 { // in case the user introduces custom pools
					validatorBatches = s.PoolValidators

					valMatrix := make([][]phase0.ValidatorIndex, len(validatorBatches))

					for i, item := range validatorBatches {
						valMatrix[i] = make([]phase0.ValidatorIndex, 0)
						valMatrix[i] = item.ValIdxs
					}

					if s.MissingVals {
						othersMissingList := utils.ObtainMissing(len(valIdxs), valMatrix)
						// now allValList should contain those validators that do not belong to any pool
						// keep track of those in a separate pool
						validatorBatches = utils.AddOthersPool(validatorBatches, othersMissingList)
					}

				}

				for _, item := range validatorBatches {
					valTask := &ValTask{
						ValIdxs:         item.ValIdxs,
						StateMetricsObj: stateMetrics,
						PoolName:        item.PoolName,
						Finalized:       task.Finalized,
					}
					s.ValTaskChan <- valTask
				}
			}
			if task.PrevState.Slot >= s.InitSlot || task.Finalized { // only write epoch metrics inside the defined range

				log.Debugf("Writing epoch metrics to DB for slot %d...", task.State.Slot)
				// create a model to be inserted into the db, we only insert previous epoch metrics

				missedBlocks := stateMetrics.GetMetricsBase().CurrentState.MissedBlocks
				// take into accoutn epoch transition
				nextMissedBlock := stateMetrics.GetMetricsBase().NextState.TrackPrevMissingBlock()
				if nextMissedBlock != 0 {
					missedBlocks = append(missedBlocks, nextMissedBlock)
				}

				// TODO: send constructor to model package
				epochModel := model.Epoch{
					Epoch:                 stateMetrics.GetMetricsBase().CurrentState.Epoch,
					Slot:                  stateMetrics.GetMetricsBase().CurrentState.Slot,
					NumAttestations:       len(stateMetrics.GetMetricsBase().NextState.PrevAttestations),
					NumAttValidators:      int(stateMetrics.GetMetricsBase().NextState.NumAttestingVals),
					NumValidators:         int(stateMetrics.GetMetricsBase().CurrentState.NumActiveVals),
					TotalBalance:          float32(stateMetrics.GetMetricsBase().CurrentState.TotalActiveRealBalance) / float32(utils.EFFECTIVE_BALANCE_INCREMENT),
					AttEffectiveBalance:   float32(stateMetrics.GetMetricsBase().NextState.AttestingBalance[altair.TimelyTargetFlagIndex]) / float32(utils.EFFECTIVE_BALANCE_INCREMENT), // as per BEaconcha.in
					TotalEffectiveBalance: float32(stateMetrics.GetMetricsBase().CurrentState.TotalActiveBalance) / float32(utils.EFFECTIVE_BALANCE_INCREMENT),
					MissingSource:         int(stateMetrics.GetMetricsBase().NextState.GetMissingFlagCount(int(altair.TimelySourceFlagIndex))),
					MissingTarget:         int(stateMetrics.GetMetricsBase().NextState.GetMissingFlagCount(int(altair.TimelyTargetFlagIndex))),
					MissingHead:           int(stateMetrics.GetMetricsBase().NextState.GetMissingFlagCount(int(altair.TimelyHeadFlagIndex)))}

				s.dbClient.WriteChan <- db.WriteTask{
					Model: epochModel,
					Op:    model.INSERT_OP,
				}

				// Proposer Duties

				for _, item := range stateMetrics.GetMetricsBase().CurrentState.EpochStructs.ProposerDuties {

					newDuty := model.ProposerDuty{
						ValIdx:       item.ValidatorIndex,
						ProposerSlot: item.Slot,
						Proposed:     true,
					}
					for _, item := range missedBlocks {
						if newDuty.ProposerSlot == item { // we found the proposer slot in the missed blocks
							newDuty.Proposed = false
						}
					}
					s.dbClient.WriteChan <- db.WriteTask{
						Model: newDuty,
						Op:    model.INSERT_OP,
					}
				}
			}
		}

	}
	log.Infof("Pre process routine finished...")
}
