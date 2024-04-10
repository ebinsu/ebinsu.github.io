@startuml
start
note left : isActive = true
if (__idleReplicaCount__ != nil && currentReplicas < __minReplicaCount__) then (yes)
	:updateScaleOnScaleTarget -> max( __minReplicaCount__ , 1);
elseif (currentReplicas == 0 ) then (yes)
	:updateScaleOnScaleTarget -> max( __minReplicaCount__ , 1);
elseif (isError) then (yes)
	:ScaledObject.Status.ReadyCondition -> Unknown;
else
	:update ScaledObject.LastActiveTime to now;
endif
stop
@enduml