package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	cart "github.com/bufengmobuganhuo/micro-service-cart/proto/cart"
	cartApi "github.com/bufengmobuganhuo/micro-service-cartApi/proto/cartApi"
	log "github.com/micro/go-micro/v2/logger"
	"strconv"
)

type CartApi struct {
	CartService cart.CartService
}

// FindAll caCartApi.FindAll 通过API向外暴露为/cartApi/findAll，接收http请求
// 即：/cartApi/findAll请求会调用go.micro.api.cartApi 服务的CartApi.FindAll方法
func (e *CartApi) FindAll(ctx context.Context, req *cartApi.Request, rsp *cartApi.Response) error {
	log.Info("接收到/cartApi/findAll访问请求")
	if _, ok := req.Get["user_id"]; !ok {
		rsp.StatusCode = 400
		return errors.New("缺少参数")
	}
	userIdString := req.Get["user_id"].Values[0]
	fmt.Println(userIdString)
	userId, err := strconv.ParseInt(userIdString, 10, 64)
	if err != nil {
		return err
	}
	// 获取购物车所有商品
	cartAll, err := e.CartService.GetAll(context.TODO(), &cart.CartFindAll{userId})
	if err != nil {
		return err
	}

	b, err := json.Marshal(cartAll)
	if err != nil {
		return err
	}

	rsp.StatusCode = 200
	rsp.Body = string(b)
	return nil
}
