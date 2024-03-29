// Code generated by "stringer -output=string.go -type=SubType,Condition,Feature -linecomment"; DO NOT EDIT.

package pubsub

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[SubNone-0]
	_ = x[SubPending-1]
	_ = x[SubSubscribed-2]
	_ = x[SubUnconfigured-3]
}

const _SubType_name = "nonependingsubscribedunconfigured"

var _SubType_index = [...]uint8{0, 4, 11, 21, 33}

func (i SubType) String() string {
	if i >= SubType(len(_SubType_index)-1) {
		return "SubType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _SubType_name[_SubType_index[i]:_SubType_index[i+1]]
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[CondNone-0]
	_ = x[CondClosedNode-1]
	_ = x[CondConfigRequired-2]
	_ = x[CondInvalidJID-3]
	_ = x[CondInvalidOptions-4]
	_ = x[CondInvalidPayload-5]
	_ = x[CondInvalidSubID-6]
	_ = x[CondItemForbidden-7]
	_ = x[CondItemRequired-8]
	_ = x[CondJIDRequired-9]
	_ = x[CondMaxItemsExceeded-10]
	_ = x[CondMaxNodesExceeded-11]
	_ = x[CondNodeIDRequired-12]
	_ = x[CondNotInRosterGroup-13]
	_ = x[CondNotSubscribed-14]
	_ = x[CondPayloadTooBig-15]
	_ = x[CondPayloadRequired-16]
	_ = x[CondPendingSubscription-17]
	_ = x[CondPresenceRequired-18]
	_ = x[CondSubIDRequired-19]
	_ = x[CondTooManySubscriptions-20]
	_ = x[CondUnsupported-21]
	_ = x[CondUnsupportedAccessModel-22]
}

const _Condition_name = "CondNoneclosed-nodeconfiguration-requiredinvalid-jidinvalid-optionsinvalid-payloadinvalid-subiditem-forbiddenitem-requiredjid-requiredmax-items-exceededmax-nodes-exceedednodeid-requirednot-in-roster-groupnot-subscribedpayload-too-bigpayload-requiredpending-subscriptionpresence-subscription-requiredsubid-requiredtoo-many-subscriptionsunsupportedunsupported-access-model"

var _Condition_index = [...]uint16{0, 8, 19, 41, 52, 67, 82, 95, 109, 122, 134, 152, 170, 185, 204, 218, 233, 249, 269, 299, 313, 335, 346, 370}

func (i Condition) String() string {
	if i >= Condition(len(_Condition_index)-1) {
		return "Condition(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Condition_name[_Condition_index[i]:_Condition_index[i+1]]
}
func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[FeatureAccessAuthorize-0]
	_ = x[FeatureAccessOpen-1]
	_ = x[FeatureAccessPresence-2]
	_ = x[FeatureAccessRoster-3]
	_ = x[FeatureAccessWhitelist-4]
	_ = x[FeatureAutoCreate-5]
	_ = x[FeatureAutoSubscribe-6]
	_ = x[FeatureCollections-7]
	_ = x[FeatureConfigNode-8]
	_ = x[FeatureCreateAndConfigure-9]
	_ = x[FeatureCreateNodes-10]
	_ = x[FeatureDeleteItems-11]
	_ = x[FeatureDeleteNodes-12]
	_ = x[FeatureFilteredNotifications-13]
	_ = x[FeatureGetPending-14]
	_ = x[FeatureInstantNodes-15]
	_ = x[FeatureItemIDs-16]
	_ = x[FeatureLastPublished-17]
	_ = x[FeatureLeasedSubscription-18]
	_ = x[FeatureManageSubscriptions-19]
	_ = x[FeatureMemberAffiliation-20]
	_ = x[FeatureMetaData-21]
	_ = x[FeatureModifyAffiliations-22]
	_ = x[FeatureMultiCollection-23]
	_ = x[FeatureMultiSubscribe-24]
	_ = x[FeatureOutcastAffiliation-25]
	_ = x[FeaturePersistentItems-26]
	_ = x[FeaturePresenceNotifications-27]
	_ = x[FeaturePresenceSubscribe-28]
	_ = x[FeaturePublish-29]
	_ = x[FeaturePublishOptions-30]
	_ = x[FeaturePublishOnlyAffiliation-31]
	_ = x[FeaturePublisherAffiliation-32]
	_ = x[FeaturePurgeNodes-33]
	_ = x[FeatureRetractItems-34]
	_ = x[FeatureRetrieveAffiliations-35]
	_ = x[FeatureRetrieveDefault-36]
	_ = x[FeatureRetrieveItems-37]
	_ = x[FeatureRetrieveSubscriptions-38]
	_ = x[FeatureSubscribe-39]
	_ = x[FeatureSubscriptionOptions-40]
	_ = x[FeatureSubscriptionNotifications-41]
}

const _Feature_name = "access-authorizeaccess-openaccess-presenceaccess-rosteraccess-whitelistauto-createauto-subscribecollectionsconfig-nodecreate-and-configurecreate-nodesdelete-itemsdelete-nodesfiltered-notificationsget-pendinginstant-nodesitem-idslast-publishedleased-subscriptionmanage-subscriptionsmember-affiliationmeta-datamodify-affiliationsmulti-collectionmulti-subscribeoutcast-affiliationpersistent-itemspresence-notificationspresence-subscribepublishpublish-optionspublish-only-affiliationpublisher-affiliationpurge-nodesretract-itemsretrieve-affiliationsretrieve-defaultretrieve-itemsretrieve-subscriptionssubscribesubscription-optionssubscription-notifications"

var _Feature_index = [...]uint16{0, 16, 27, 42, 55, 71, 82, 96, 107, 118, 138, 150, 162, 174, 196, 207, 220, 228, 242, 261, 281, 299, 308, 327, 343, 358, 377, 393, 415, 433, 440, 455, 479, 500, 511, 524, 545, 561, 575, 597, 606, 626, 652}

func (i Feature) String() string {
	if i >= Feature(len(_Feature_index)-1) {
		return "Feature(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _Feature_name[_Feature_index[i]:_Feature_index[i+1]]
}
