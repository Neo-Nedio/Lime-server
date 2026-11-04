// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package handler

import (
	"net/http"

	"github.com/Neo-Nedio/Lime-server/auth/api/internal/logic"
	"github.com/Neo-Nedio/Lime-server/auth/api/internal/svc"
	"github.com/Neo-Nedio/Lime-server/auth/api/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func RefreshHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RefreshReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := logic.NewRefreshLogic(r.Context(), svcCtx)
		resp, err := l.Refresh(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
