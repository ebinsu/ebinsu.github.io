@startuml
start
note left : isActive = false
if (isError) then (yes)
	if (fallback.replicas != 0) then (yes)
		:updateScaleOnScaleTarget -> fallback.replicas;
	else (no)
		:ScaledObject.Status.ReadyCondition -> False;
	endif
else (no)
	if (idleReplicaCount != nil && currentReplicas > idleReplicaCount) then (yes)
		if (ScaledObject.LastActiveTime == nil || ScaledObject.LastActiveTime.Add(cooldownPeriod).Before(now)) then (yes)
			:updateScaleOnScaleTarget -> idleReplicaCount;
		endif
	elseif (currentReplicas > 0 && minReplicas == 0) then (yes)
		if (ScaledObject.LastActiveTime == nil || LastActiveTime.Add(cooldownPeriod).Before(now)) then (yes)
			:updateScaleOnScaleTarget -> 0;
		endif
	elseif (currentReplicas < minReplicaCount && idleReplicaCount == nil) then (yes)
		:updateScaleOnScaleTarget -> minReplicaCount;
	else 
		:nothing needs to be done;
	endif
endif
stop
@enduml