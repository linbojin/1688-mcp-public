package errors

import "errors"

var (
	ErrNotLoggedIn     = errors.New("未登录1688")
	ErrSearchNoResult  = errors.New("搜索无结果")
	ErrProductNotFound = errors.New("商品不存在")
	ErrPuhuoFailed     = errors.New("铺货任务提交失败")
	ErrPageTimeout     = errors.New("页面加载超时")
)
