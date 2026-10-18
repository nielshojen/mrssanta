package eventupload

import (
	"time"

	syncv1 "buf.build/gen/go/northpolesec/protos/protocolbuffers/go/sync"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// eventFromProto maps an execution event from the sync protocol onto the
// stored event. Enums are stored by name, as they were sent by Santa.
func eventFromProto(e *syncv1.Event) Event {
	event := Event{
		FileSha256:                  e.GetFileSha256(),
		FilePath:                    e.GetFilePath(),
		FileName:                    e.GetFileName(),
		ExecutingUser:               e.GetExecutingUser(),
		ExecutionTime:               e.GetExecutionTime(),
		LoggedinUsers:               e.GetLoggedInUsers(),
		CurrentSessions:             e.GetCurrentSessions(),
		PID:                         int(e.GetPid()),
		PPID:                        int(e.GetPpid()),
		Decision:                    e.GetDecision().String(),
		FileBundleID:                e.GetFileBundleId(),
		FileBundlePath:              e.GetFileBundlePath(),
		FileBundleExecutableRelPath: e.GetFileBundleExecutableRelPath(),
		FileBundleName:              e.GetFileBundleName(),
		FileBundleVersion:           e.GetFileBundleVersion(),
		FileBundleVersionString:     e.GetFileBundleVersionString(),
		FileBundleHash:              e.GetFileBundleHash(),
		FileBundleHashMillis:        float64(e.GetFileBundleHashMillis()),
		FileBundleBinaryCount:       int(e.GetFileBundleBinaryCount()),
		ParentName:                  e.GetParentName(),
		QuarantineDataURL:           e.GetQuarantineDataUrl(),
		QuarantineRefererURL:        e.GetQuarantineRefererUrl(),
		QuarantineTimestamp:         float64(e.GetQuarantineTimestamp()),
		QuarantineAgentBundleID:     e.GetQuarantineAgentBundleId(),
		SigningChain:                signingChainFromProto(e.GetSigningChain()),
		SigningID:                   e.GetSigningId(),
		TeamID:                      e.GetTeamId(),
		CDHash:                      e.GetCdhash(),
		CSFlags:                     int64(e.GetCsFlags()),
		SecureSigningTime:           e.GetSecureSigningTime(),
		SigningTime:                 e.GetSigningTime(),
		StaticRule:                  e.GetStaticRule(),
		IdentityUnverified:          e.GetIdentityUnverified(),
	}

	if e.GetSigningStatus() != syncv1.SigningStatus_SIGNING_STATUS_UNSPECIFIED {
		event.SigningStatus = e.GetSigningStatus().String()
	}

	if e.HasEntitlementInfo() {
		info := &EntitlementInfo{EntitlementsFiltered: e.GetEntitlementInfo().GetEntitlementsFiltered()}
		for _, entitlement := range e.GetEntitlementInfo().GetEntitlements() {
			info.Entitlements = append(info.Entitlements, Entitlement{
				Key:   entitlement.GetKey(),
				Value: entitlement.GetValue(),
			})
		}
		event.EntitlementInfo = info
	}

	return event
}

func signingChainFromProto(chain []*syncv1.Certificate) []SigningChain {
	if len(chain) == 0 {
		return nil
	}

	result := make([]SigningChain, 0, len(chain))
	for _, cert := range chain {
		result = append(result, SigningChain{
			Sha256:     cert.GetSha256(),
			CN:         cert.GetCn(),
			Org:        cert.GetOrg(),
			OU:         cert.GetOu(),
			ValidFrom:  int(cert.GetValidFrom()),
			ValidUntil: int(cert.GetValidUntil()),
		})
	}
	return result
}

func fileAccessEventFromProto(e *syncv1.FileAccessEvent, machineID string, now time.Time) FileAccessEvent {
	event := FileAccessEvent{
		MachineID:    machineID,
		RuleName:     e.GetRuleName(),
		RuleVersion:  e.GetRuleVersion(),
		Target:       e.GetTarget(),
		Decision:     e.GetDecision().String(),
		AccessTime:   e.GetAccessTime(),
		CreationTime: primitive.NewDateTimeFromTime(now),
	}

	for _, process := range e.GetProcessChain() {
		event.ProcessChain = append(event.ProcessChain, Process{
			FilePath:     process.GetFilePath(),
			CDHash:       process.GetCdhash(),
			FileSHA256:   process.GetFileSha256(),
			SigningID:    process.GetSigningId(),
			TeamID:       process.GetTeamId(),
			PID:          process.GetPid(),
			SigningChain: signingChainFromProto(process.GetSigningChain()),
		})
	}

	return event
}

// standaloneApprovalsFromProto extracts the STANDALONE mode rule creations
// from a batch of audit events. Other audit event types are ignored.
func standaloneApprovalsFromProto(events []*syncv1.AuditEvent, machineID string, now time.Time) []StandaloneApproval {
	var approvals []StandaloneApproval
	for _, event := range events {
		if !event.HasStandaloneModeRuleCreation() {
			continue
		}

		creation := event.GetStandaloneModeRuleCreation()
		approval := StandaloneApproval{
			MachineID:    machineID,
			Identifier:   creation.GetIdentifier(),
			Decision:     creation.GetDecision().String(),
			CreationTime: primitive.NewDateTimeFromTime(now),
		}
		if creation.GetTimestamp() != 0 {
			approval.Timestamp = primitive.NewDateTimeFromTime(time.Unix(int64(creation.GetTimestamp()), 0).UTC())
		}
		approvals = append(approvals, approval)
	}
	return approvals
}
