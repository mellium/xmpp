// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:generate go run -tags=tools golang.org/x/tools/cmd/stringer -type=Affiliation,Role,Privileges -linecomment

package muc

import (
	"encoding/xml"
	"errors"
)

// Affiliation indicates a users affiliation to the room.
type Affiliation uint8

// A list of room affiliations.
const (
	AffiliationNone Affiliation = iota // none

	// Support for the owner affiliation is required.
	AffiliationOwner // owner

	// Support for these affiliations is recommended, but optional.
	AffiliationAdmin   // admin
	AffiliationMember  // member
	AffiliationOutcast // outcast
)

// UnmarshalXMLAttr satisfies xml.UnmarshalerAttr.
func (a *Affiliation) UnmarshalXMLAttr(attr xml.Attr) error {
	switch attr.Value {
	case AffiliationNone.String():
		*a = AffiliationNone
	case AffiliationOwner.String():
		*a = AffiliationOwner
	case AffiliationAdmin.String():
		*a = AffiliationAdmin
	case AffiliationMember.String():
		*a = AffiliationMember
	case AffiliationOutcast.String():
		*a = AffiliationOutcast
	default:
		return errors.New("muc: unrecognized affiliation")
	}
	return nil
}

// MarshalXMLAttr satisfies xml.MarshalerAttr.
func (a *Affiliation) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	return xml.Attr{Name: name, Value: a.String()}, nil
}

// Role indicates a users role in the room.
type Role uint8

// A list of user roles.
const (
	RoleNone Role = iota // none

	// Support for these roles is required.
	RoleModerator   // moderator
	RoleParticipant // participant

	// Support for these roles is recommended, but optional.
	RoleVisitor // visitor
)

// UnmarshalXMLAttr satisfies xml.UnmarshalerAttr.
func (r *Role) UnmarshalXMLAttr(attr xml.Attr) error {
	switch attr.Value {
	case RoleNone.String():
		*r = RoleNone
	case RoleModerator.String():
		*r = RoleModerator
	case RoleParticipant.String():
		*r = RoleParticipant
	case RoleVisitor.String():
		*r = RoleVisitor
	default:
		return errors.New("muc: unrecognized role")
	}
	return nil
}

// MarshalXMLAttr satisfies xml.MarshalerAttr.
func (r *Role) MarshalXMLAttr(name xml.Name) (xml.Attr, error) {
	if r == nil {
		return xml.Attr{}, nil
	}
	return xml.Attr{Name: name, Value: r.String()}, nil
}

// Privileges is a bit mask indicating the various privileges assigned to a room
// user.
type Privileges uint16

// A list of possible privileges.
const (
	PrivilegePresent            Privileges = 1 << iota // present
	PrivilegeReceiveMessages                           // receive-messages
	PrivilegeReceivePresence                           // receive-presence
	PrivilegeBroadcastPresence                         // broadcast-presence
	PrivilegeChangeAvailability                        // change-availability
	PrivilegeChangeNick                                // change-nick
	PrivilegePrivateMessage                            // send-private-message
	PrivilegeSendInvites                               // send-invites
	PrivilegeSendMessages                              // send-messages
	PrivilegeModifySubject                             // modify-subject
	PrivilegeKick                                      // kick
	PrivilegeGrantVoice                                // grant-voice
	PrivilegeRevokeVoice                               // revoke-voice

	// Common default privilages for each role.
	// These are just common defaults provided as a convenience, it is not
	// guaranteed that a user of a given role has this set of privileges.
	PrivilegesVisitor     = PrivilegePresent | PrivilegeReceiveMessages | PrivilegeReceivePresence | PrivilegeBroadcastPresence | PrivilegeChangeAvailability | PrivilegeChangeNick | PrivilegePrivateMessage | PrivilegeSendInvites
	PrivilegesParticipant = PrivilegesVisitor | PrivilegeSendMessages | PrivilegeModifySubject
	PrivilegesModerator   = PrivilegesParticipant | PrivilegeKick | PrivilegeGrantVoice | PrivilegeRevokeVoice
)
