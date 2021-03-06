package newshttpdelivery

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo"
	"github.com/pkg/errors"
	"gitlab.com/99ridho/news-api/domain/news"
	"gitlab.com/99ridho/news-api/models"
)

type NewsMutationHandler func(ctx context.Context, n *models.News) (*models.News, error)

type NewsHandler struct {
	UseCase news.NewsUseCase
}

func (h *NewsHandler) convertTopicIDParam(param string) ([]int64, error) {
	topicIDs := make([]int64, 0)
	if param != "" {
		ids := strings.Split(param, ",")
		for _, strID := range ids {
			num, err := strconv.Atoi(strID)
			if err != nil {
				return nil, errors.Wrap(err, "Request data invalid")
			}
			topicIDs = append(topicIDs, int64(num))
		}
	}
	return topicIDs, nil
}

func (h *NewsHandler) mutateNews(c echo.Context, mutationHandler NewsMutationHandler) error {
	req := new(MutateNewsRequest)
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusBadRequest, &models.GeneralResponse{
			Data:         nil,
			ErrorMessage: errors.Wrap(err, "Request data invalid").Error(),
			Message:      "Fail",
		})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	news := req.News
	news.TopicIDs = req.NewsTopic

	result, err := mutationHandler(ctx, news)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &models.GeneralResponse{
			Data:         nil,
			ErrorMessage: err.Error(),
			Message:      "Fail",
		})
	}

	return c.JSON(200, &models.GeneralResponse{
		Data: &MutateNewsResponse{
			News: result,
		},
		ErrorMessage: "",
		Message:      "OK",
	})
}

func (h *NewsHandler) FetchNews(c echo.Context) error {
	params := new(FetchNewsRequest)
	if err := c.Bind(params); err != nil {
		return c.JSON(http.StatusBadRequest, &models.GeneralResponse{
			Data:         nil,
			ErrorMessage: errors.Wrap(err, "Request data invalid").Error(),
			Message:      "Fail",
		})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	fetchParams := &models.FetchNewsParam{
		Pagination: &models.Pagination{Limit: params.Limit, NextCursor: params.NextCursor},
		Status:     params.Status,
	}

	topicIDs, err := h.convertTopicIDParam(params.Topic)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &models.GeneralResponse{
			Data:         nil,
			ErrorMessage: err.Error(),
			Message:      "Fail",
		})
	}
	fetchParams.TopicIDs = topicIDs

	result, pagination, err := h.UseCase.FetchNewsByParams(ctx, fetchParams)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &models.GeneralResponse{
			Data:         nil,
			ErrorMessage: err.Error(),
			Message:      "Fail",
		})
	}

	return c.JSON(200, &models.GeneralResponse{
		Data: &NewsResponse{
			News:       result,
			Pagination: pagination,
		},
		ErrorMessage: "",
		Message:      "OK",
	})
}

func (h *NewsHandler) InsertNews(c echo.Context) error {
	return h.mutateNews(c, func(ctx context.Context, n *models.News) (*models.News, error) {
		n.Status = "draft"
		return h.UseCase.InsertNews(ctx, n)
	})
}

func (h *NewsHandler) UpdateNews(c echo.Context) error {
	return h.mutateNews(c, func(ctx context.Context, n *models.News) (*models.News, error) {
		id := c.Param("id")
		intId, err := strconv.Atoi(id)
		if err != nil {
			return nil, errors.Wrap(err, "id must integer")
		}

		n.ID = int64(intId)
		return h.UseCase.UpdateNews(ctx, n)
	})
}

func (h *NewsHandler) DeleteNews(c echo.Context) error {
	id := c.Param("id")
	intId, err := strconv.Atoi(id)
	if err != nil {
		return c.JSON(http.StatusBadRequest, &models.GeneralResponse{
			Data:         nil,
			ErrorMessage: errors.Wrap(err, "Topic ID must int").Error(),
			Message:      "Fail",
		})
	}

	ctx := c.Request().Context()
	if ctx == nil {
		ctx = context.Background()
	}

	ok, err := h.UseCase.DeleteNews(ctx, int64(intId))
	if err != nil {
		return c.JSON(http.StatusInternalServerError, &models.GeneralResponse{
			Data:         nil,
			ErrorMessage: errors.Wrap(err, "Delete topic failed").Error(),
			Message:      "Fail",
		})
	}

	return c.JSON(http.StatusOK, &models.GeneralResponse{
		Data: &DeleteNewsResponse{
			IsSuccess: ok,
		},
		ErrorMessage: "",
		Message:      "OK",
	})
}

func InitializeNewsHandler(r *echo.Echo, usecase news.NewsUseCase) {
	handler := &NewsHandler{usecase}

	g := r.Group("/news")

	g.GET("", handler.FetchNews)
	g.POST("", handler.InsertNews)
	g.PUT("/:id", handler.UpdateNews)
	g.DELETE("/:id", handler.DeleteNews)
}
