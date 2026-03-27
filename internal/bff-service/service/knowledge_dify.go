package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	http_client "github.com/UnicomAI/wanwu/pkg/http-client"
	"github.com/UnicomAI/wanwu/pkg/log"
)

type DifyKnowledgeRetrieveParams struct {
	Query          string              `json:"query" validate:"required"`
	RetrievalModel *DifyRetrievalModel `json:"retrieval_model" validate:"required"`
}

type DifyRetrievalModel struct {
	SearchMethod                string                           `json:"search_method"`
	RerankingEnable             bool                             `json:"reranking_enable"`
	RerankingMode               string                           `json:"reranking_mode"`
	RerankingModel              *DifyRerankingModel              `json:"reranking_model"`
	TopK                        int64                            `json:"top_k"`
	ScoreThresholdEnabled       bool                             `json:"score_threshold_enabled"`
	ScoreThreshold              float32                          `json:"score_threshold"`
	Weights                     *DifyWeights                     `json:"weights"`
	MetadataFilteringConditions *DifyMetadataFilteringConditions `json:"metadata_filtering_conditions"`
}

type DifyWeights struct {
	WeightType     string              `json:"weight_type"`
	KeywordSetting *DifyKeywordSetting `json:"keyword_setting"`
	VectorSetting  *DifyVectorSetting  `json:"vector_setting"`
}

type DifyKeywordSetting struct {
	KeywordWeight float32 `json:"keyword_weight"`
}

type DifyVectorSetting struct {
	VectorWeight          float32 `json:"vector_weight"`
	EmbeddingModelName    string  `json:"embedding_model_name"`
	EmbeddingProviderName string  `json:"embedding_provider_name"`
}

type DifyRerankingModel struct {
	RerankingProviderName string `json:"reranking_provider_name"`
	RerankingModelName    string `json:"reranking_model_name"`
}

type DifyMetadataFilteringConditions struct {
	LogicalOperator string                   `json:"logical_operator"`
	Conditions      []*DifyMetadataCondition `json:"conditions"`
}

type DifyMetadataCondition struct {
	Name               string `json:"name"`
	ComparisonOperator string `json:"comparison_operator"`
	Value              string `json:"value"`
}

type DifyKnowledgeRetrieveResp struct {
	Query   *DifyQueryContent          `json:"query"`
	Records []*DifyKnowledgeHitRecords `json:"records"`
}

type DifyQueryContent struct {
	Content string `json:"content"`
}

type DifyKnowledgeHitRecords struct {
	Segment     *DifySegment      `json:"segment"`
	Score       float64           `json:"score"`
	ChildChunks []*DifyChildChunk `json:"child_chunks"`
}

type DifyChildChunk struct {
	Id       string  `json:"id"`
	Content  string  `json:"content"`
	Position int64   `json:"position"`
	Score    float64 `json:"score"`
}

type DifySegment struct {
	Id            string        `json:"id"`
	Position      int64         `json:"position"`
	DocumentId    string        `json:"document_id"`
	Content       string        `json:"content"`
	Answer        string        `json:"answer"`
	WordCount     int64         `json:"word_count"`
	Tokens        int64         `json:"tokens"`
	Keywords      []string      `json:"keywords"`
	IndexNodeId   string        `json:"index_node_id"`
	IndexNodeHash string        `json:"index_node_hash"`
	HitCount      int64         `json:"hit_count"`
	Enabled       bool          `json:"enabled"`
	DisabledAt    int64         `json:"disabled_at"`
	DisabledBy    string        `json:"disabled_by"`
	Status        string        `json:"status"`
	CreatedAt     int64         `json:"created_at"`
	CreatedBy     string        `json:"created_by"`
	IndexingAt    int64         `json:"indexing_at"`
	CompletedAt   int64         `json:"completed_at"`
	Error         string        `json:"error"`
	StoppedAt     int64         `json:"stopped_at"`
	Document      *DifyDocument `json:"document"`
}

type DifyDocument struct {
	Id             string `json:"id"`
	DataSourceType string `json:"data_source_type"`
	Name           string `json:"name"`
}

func DifyKnowledgeRetrieve(ctx context.Context, endpoint, apiKey, externalKnowledgeId string, knowledgeRetrieveParams *DifyKnowledgeRetrieveParams) (*DifyKnowledgeRetrieveResp, error) {
	knowledgeConfig := config.Cfg().DifyKnowledgeConfig
	url := strings.TrimRight(endpoint, "/") + strings.ReplaceAll(knowledgeConfig.SearchKnowledgeBaseUri, "{dataset_id}", externalKnowledgeId)
	headers := map[string]string{}
	headers["Authorization"] = fmt.Sprintf("Bearer %s", apiKey)
	paramsByte, err := json.Marshal(knowledgeRetrieveParams)
	if err != nil {
		return nil, err
	}
	result, err := ragKnowHttp.PostJson(ctx, &http_client.HttpRequestParams{
		Url:        url,
		Headers:    headers,
		Body:       paramsByte,
		Timeout:    time.Duration(knowledgeConfig.SearchKnowTimeout) * time.Second,
		MonitorKey: "dify_knowledge_retrieve",
		LogLevel:   http_client.LogAll,
	})
	if err != nil {
		return nil, err
	}
	var resp DifyKnowledgeRetrieveResp
	if err := json.Unmarshal(result, &resp); err != nil {
		log.Errorf(err.Error())
		return nil, err
	}
	return &resp, nil
}
