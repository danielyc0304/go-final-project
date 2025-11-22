package controllers

import (
	"backend/models"
	"backend/services"
	"backend/utils"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	beego "github.com/beego/beego/v2/server/web"
)

type AuthController struct {
	beego.Controller
}

// URLMapping ...
func (c *AuthController) URLMapping() {
	c.Mapping("Registration", c.Registration)
	c.Mapping("Login", c.Login)
	c.Mapping("GetOne", c.GetOne)
	c.Mapping("GetAll", c.GetAll)
	c.Mapping("Put", c.Put)
	c.Mapping("Delete", c.Delete)
}

// Registration ...
// @Title Registration
// @Description user registration
// @Param body body models.Registration	true " "
// @Success 201 {object} utils.APIResponse
// @Failure 400 missing or invalid fields
// @Failure 500 internal error
// @router /registration [post]
func (c *AuthController) Registration() {
	m := models.Registration{}

	json.Unmarshal(c.Ctx.Input.RequestBody, &m)
	if ok, errors, err := utils.ValidateField(m); err != nil {
		utils.CreateAPIResponse(c.Ctx, 500, "validation internal error: "+err.Error())
		return
	} else if !ok {
		utils.CreateAPIResponse(c.Ctx, 400, errors)
		return
	}

	id, err := services.Registration(m)
	if err != nil && err.Error() == "user already registered" {
		utils.CreateAPIResponse(c.Ctx, 409, err.Error())
		return
	} else if err != nil {
		utils.CreateAPIResponse(c.Ctx, 500, "service internal error: "+err.Error())
		return
	}
	utils.CreateAPIResponse(c.Ctx, 201, "user created successfully with id: "+strconv.FormatInt(id, 10))
}

// Login ...
// @Title Login
// @Description user login
// @Param body body models.Login true " "
// @Success 200 {object} utils.APIResponse
// @Failure 400 missing or invalid fields
// @Failure 401 invalid email or password
// @Failure 500 internal error
// @router /login [post]
func (c *AuthController) Login() {
	m := models.Login{}

	json.Unmarshal(c.Ctx.Input.RequestBody, &m)
	if ok, errors, err := utils.ValidateField(m); err != nil {
		utils.CreateAPIResponse(c.Ctx, 500, "validation internal error: "+err.Error())
		return
	} else if !ok {
		utils.CreateAPIResponse(c.Ctx, 400, errors)
		return
	}

	token, expiredAt, err := services.Login(m)
	if err != nil && err.Error() == "invalid email or password" {
		utils.CreateAPIResponse(c.Ctx, 401, err.Error())
		return
	} else if err != nil {
		utils.CreateAPIResponse(c.Ctx, 500, "service internal error: "+err.Error())
		return
	}
	utils.CreateAPIResponse(c.Ctx, 200, map[string]string{"token": token, "expiredAt": expiredAt.Format(time.RFC3339)})
}

// GetOne ...
// @Title Get One
// @Description get Auth by id
// @Param	id		path 	string	true		"The key for staticblock"
// @Success 200 {object} models.Auth
// @Failure 403 :id is empty
// @router /:id [get]
func (c *AuthController) GetOne() {
	idStr := c.Ctx.Input.Param(":id")
	id, _ := strconv.ParseInt(idStr, 0, 64)
	v, err := models.GetAuthById(id)
	if err != nil {
		c.Data["json"] = err.Error()
	} else {
		c.Data["json"] = v
	}
	c.ServeJSON()
}

// GetAll ...
// @Title Get All
// @Description get Auth
// @Param	query	query	string	false	"Filter. e.g. col1:v1,col2:v2 ..."
// @Param	fields	query	string	false	"Fields returned. e.g. col1,col2 ..."
// @Param	sortby	query	string	false	"Sorted-by fields. e.g. col1,col2 ..."
// @Param	order	query	string	false	"Order corresponding to each sortby field, if single value, apply to all sortby fields. e.g. desc,asc ..."
// @Param	limit	query	string	false	"Limit the size of result set. Must be an integer"
// @Param	offset	query	string	false	"Start position of result set. Must be an integer"
// @Success 200 {object} models.Auth
// @Failure 403
// @router / [get]
func (c *AuthController) GetAll() {
	var fields []string
	var sortby []string
	var order []string
	var query = make(map[string]string)
	var limit int64 = 10
	var offset int64

	// fields: col1,col2,entity.col3
	if v := c.GetString("fields"); v != "" {
		fields = strings.Split(v, ",")
	}
	// limit: 10 (default is 10)
	if v, err := c.GetInt64("limit"); err == nil {
		limit = v
	}
	// offset: 0 (default is 0)
	if v, err := c.GetInt64("offset"); err == nil {
		offset = v
	}
	// sortby: col1,col2
	if v := c.GetString("sortby"); v != "" {
		sortby = strings.Split(v, ",")
	}
	// order: desc,asc
	if v := c.GetString("order"); v != "" {
		order = strings.Split(v, ",")
	}
	// query: k:v,k:v
	if v := c.GetString("query"); v != "" {
		for _, cond := range strings.Split(v, ",") {
			kv := strings.SplitN(cond, ":", 2)
			if len(kv) != 2 {
				c.Data["json"] = errors.New("error: invalid query key/value pair")
				c.ServeJSON()
				return
			}
			k, v := kv[0], kv[1]
			query[k] = v
		}
	}

	l, err := models.GetAllAuth(query, fields, sortby, order, offset, limit)
	if err != nil {
		c.Data["json"] = err.Error()
	} else {
		c.Data["json"] = l
	}
	c.ServeJSON()
}

// Put ...
// @Title Put
// @Description update the Auth
// @Param	id		path 	string	true		"The id you want to update"
// @Param	body		body 	models.Auth	true		"body for Auth content"
// @Success 200 {object} models.Auth
// @Failure 403 :id is not int
// @router /:id [put]
func (c *AuthController) Put() {
	idStr := c.Ctx.Input.Param(":id")
	id, _ := strconv.ParseInt(idStr, 0, 64)
	v := models.Auth{Id: id}
	json.Unmarshal(c.Ctx.Input.RequestBody, &v)
	if err := models.UpdateAuthById(&v); err == nil {
		c.Data["json"] = "OK"
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJSON()
}

// Delete ...
// @Title Delete
// @Description delete the Auth
// @Param	id		path 	string	true		"The id you want to delete"
// @Success 200 {string} delete success!
// @Failure 403 id is empty
// @router /:id [delete]
func (c *AuthController) Delete() {
	idStr := c.Ctx.Input.Param(":id")
	id, _ := strconv.ParseInt(idStr, 0, 64)
	if err := models.DeleteAuth(id); err == nil {
		c.Data["json"] = "OK"
	} else {
		c.Data["json"] = err.Error()
	}
	c.ServeJSON()
}
