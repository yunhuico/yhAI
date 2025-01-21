package payload

import "jihulab.com/jihulab/ultrafox/ultrafox/pkg/model"

type ChangeOrgMemberRoleReq struct {
	NewRole model.OrgMemberRole `json:"newRole" binding:"required"`
}

type RemoveMembersFromOrgReq struct {
	UserIDs []int `json:"userIds" binding:"required,min=1"`
}

type LeaveOrgReq struct {
	SuccessorUserID int `json:"successorUserId"`
}

type ListOrgMembersByIDsReq struct {
	UserIDs []int `json:"userIds" binding:"required,min=1,max=20"`
}

type SearchOrgMembersReq struct {
	Query string `json:"q" form:"q"`
	Size  int    `json:"size" form:"size" binding:"min=0,max=10"`
}
