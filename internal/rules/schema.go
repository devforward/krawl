package rules

import (
	"fmt"
	"strings"

	"github.com/devforward/krawl/internal/parser"
)

// schemaRequirements defines required and recommended properties per schema.org @type.
// Required properties trigger errors when missing; recommended trigger warnings.
type schemaReq struct {
	Required    []string
	Recommended []string
}

var schemaRequirements = map[string]schemaReq{
	"Article": {
		Required:    []string{"headline", "datePublished"},
		Recommended: []string{"author", "image", "dateModified", "publisher"},
	},
	"NewsArticle": {
		Required:    []string{"headline", "datePublished"},
		Recommended: []string{"author", "image", "dateModified", "publisher"},
	},
	"BlogPosting": {
		Required:    []string{"headline", "datePublished"},
		Recommended: []string{"author", "image", "dateModified", "publisher"},
	},
	"Product": {
		Required:    []string{"name"},
		Recommended: []string{"image", "description", "offers", "aggregateRating", "review", "brand", "sku"},
	},
	"LocalBusiness": {
		Required:    []string{"name", "address"},
		Recommended: []string{"telephone", "openingHours", "geo", "image", "url"},
	},
	"Organization": {
		Required:    []string{"name"},
		Recommended: []string{"url", "logo", "sameAs"},
	},
	"Person": {
		Required:    []string{"name"},
		Recommended: []string{"url", "image", "sameAs", "jobTitle"},
	},
	"FAQPage": {
		Required:    []string{"mainEntity"},
		Recommended: []string{},
	},
	"HowTo": {
		Required:    []string{"name", "step"},
		Recommended: []string{"description", "image", "totalTime"},
	},
	"BreadcrumbList": {
		Required:    []string{"itemListElement"},
		Recommended: []string{},
	},
	"WebSite": {
		Required:    []string{},
		Recommended: []string{"name", "url", "potentialAction"},
	},
	"WebPage": {
		Required:    []string{},
		Recommended: []string{"name", "url", "description"},
	},
	"Event": {
		Required:    []string{"name", "startDate", "location"},
		Recommended: []string{"description", "image", "endDate", "offers", "organizer"},
	},
	"Recipe": {
		Required:    []string{"name"},
		Recommended: []string{"image", "author", "datePublished", "description", "recipeIngredient", "recipeInstructions"},
	},
	"VideoObject": {
		Required:    []string{"name", "description", "thumbnailUrl", "uploadDate"},
		Recommended: []string{"duration", "contentUrl", "embedUrl"},
	},
}

// EvaluateSchema runs only schema validation and rich result eligibility checks.
func EvaluateSchema(data *parser.SEOData) []Result {
	var results []Result
	results = append(results, checkSchemaValidation(data)...)
	results = append(results, checkRichResults(data)...)
	return results
}

func checkSchemaValidation(data *parser.SEOData) []Result {
	var results []Result
	cat := "Schema Validation"

	if len(data.JSONLDBlocks) == 0 {
		return results
	}

	// Collect all typed objects (top-level and from @graph)
	var objects []map[string]interface{}
	for _, block := range data.JSONLDBlocks {
		if graph, ok := block["@graph"].([]interface{}); ok {
			for _, item := range graph {
				if obj, ok := item.(map[string]interface{}); ok {
					objects = append(objects, obj)
				}
			}
		} else {
			objects = append(objects, block)
		}
	}

	validated := 0
	for _, obj := range objects {
		typ := getType(obj)
		if typ == "" {
			continue
		}

		req, known := schemaRequirements[typ]
		if !known {
			continue
		}
		validated++

		// Check required properties
		for _, prop := range req.Required {
			if !hasProperty(obj, prop) {
				results = append(results, Result{cat, fmt.Sprintf("%s.%s", typ, prop), SeverityError,
					fmt.Sprintf("%s is missing required property %q", typ, prop)})
			}
		}

		// Check recommended properties
		for _, prop := range req.Recommended {
			if !hasProperty(obj, prop) {
				results = append(results, Result{cat, fmt.Sprintf("%s.%s", typ, prop), SeverityWarning,
					fmt.Sprintf("%s is missing recommended property %q", typ, prop)})
			}
		}

		// Type-specific deep validation
		results = append(results, validateTypeSpecific(obj, typ)...)
	}

	if validated > 0 {
		results = append([]Result{{cat, "Schema types validated", SeverityPass,
			fmt.Sprintf("Validated %d schema type(s)", validated)}}, results...)
	}

	return results
}

func checkRichResults(data *parser.SEOData) []Result {
	var results []Result

	if len(data.JSONLDBlocks) == 0 {
		return results
	}

	// Collect all typed objects
	var objects []map[string]interface{}
	for _, block := range data.JSONLDBlocks {
		if graph, ok := block["@graph"].([]interface{}); ok {
			for _, item := range graph {
				if obj, ok := item.(map[string]interface{}); ok {
					objects = append(objects, obj)
				}
			}
		} else {
			objects = append(objects, block)
		}
	}

	for _, obj := range objects {
		typ := getType(obj)
		switch typ {
		case "FAQPage":
			results = append(results, checkFAQRichResult(obj)...)
		case "HowTo":
			results = append(results, checkHowToRichResult(obj)...)
		case "Article", "NewsArticle", "BlogPosting":
			results = append(results, checkArticleRichResult(obj, typ)...)
		case "Product":
			results = append(results, checkProductRichResult(obj)...)
		case "BreadcrumbList":
			results = append(results, checkBreadcrumbRichResult(obj)...)
		}
	}

	return results
}

func checkFAQRichResult(obj map[string]interface{}) []Result {
	var results []Result
	cat := "Rich Results"

	mainEntity, ok := obj["mainEntity"]
	if !ok {
		return results
	}

	items, ok := mainEntity.([]interface{})
	if !ok {
		results = append(results, Result{cat, "FAQ mainEntity", SeverityError,
			"FAQPage mainEntity should be an array of Question objects"})
		return results
	}

	if len(items) == 0 {
		results = append(results, Result{cat, "FAQ questions", SeverityError,
			"FAQPage has empty mainEntity array"})
		return results
	}

	validQuestions := 0
	for i, item := range items {
		q, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		qType := getType(q)
		if qType != "Question" {
			results = append(results, Result{cat, fmt.Sprintf("FAQ item #%d", i+1), SeverityError,
				fmt.Sprintf("FAQ item should be @type Question, got %q", qType)})
			continue
		}
		if !hasProperty(q, "name") {
			results = append(results, Result{cat, fmt.Sprintf("FAQ Q#%d name", i+1), SeverityError,
				"FAQ Question is missing name (the question text)"})
		}
		if answer, ok := q["acceptedAnswer"].(map[string]interface{}); ok {
			if !hasProperty(answer, "text") {
				results = append(results, Result{cat, fmt.Sprintf("FAQ Q#%d answer.text", i+1), SeverityError,
					"FAQ acceptedAnswer is missing text"})
			} else {
				validQuestions++
			}
		} else {
			results = append(results, Result{cat, fmt.Sprintf("FAQ Q#%d acceptedAnswer", i+1), SeverityError,
				"FAQ Question is missing acceptedAnswer"})
		}
	}

	if validQuestions > 0 {
		results = append([]Result{{cat, "FAQ rich result", SeverityPass,
			fmt.Sprintf("Eligible: %d valid Q&A pair(s)", validQuestions)}}, results...)
	}

	return results
}

func checkHowToRichResult(obj map[string]interface{}) []Result {
	var results []Result
	cat := "Rich Results"

	steps, ok := obj["step"]
	if !ok {
		results = append(results, Result{cat, "HowTo steps", SeverityError,
			"HowTo is missing step property (required for rich result)"})
		return results
	}

	stepList, ok := steps.([]interface{})
	if !ok {
		results = append(results, Result{cat, "HowTo steps", SeverityError,
			"HowTo step should be an array"})
		return results
	}

	validSteps := 0
	for i, item := range stepList {
		step, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		hasText := hasProperty(step, "text")
		hasName := hasProperty(step, "name")
		if !hasText && !hasName {
			results = append(results, Result{cat, fmt.Sprintf("HowTo step #%d", i+1), SeverityError,
				"HowTo step needs text or name property"})
		} else {
			validSteps++
		}
	}

	if validSteps > 0 {
		results = append([]Result{{cat, "HowTo rich result", SeverityPass,
			fmt.Sprintf("Eligible: %d valid step(s)", validSteps)}}, results...)
	}

	return results
}

func checkArticleRichResult(obj map[string]interface{}, typ string) []Result {
	var results []Result
	cat := "Rich Results"

	eligible := true

	if !hasProperty(obj, "headline") {
		eligible = false
	}
	if !hasProperty(obj, "datePublished") {
		eligible = false
	}
	if !hasProperty(obj, "image") {
		results = append(results, Result{cat, typ + " image", SeverityWarning,
			fmt.Sprintf("%s missing image (required for Article rich result)", typ)})
		eligible = false
	}

	if publisher, ok := obj["publisher"].(map[string]interface{}); ok {
		if !hasProperty(publisher, "name") {
			results = append(results, Result{cat, typ + " publisher.name", SeverityWarning,
				"Article publisher is missing name"})
		}
		if !hasProperty(publisher, "logo") {
			results = append(results, Result{cat, typ + " publisher.logo", SeverityWarning,
				"Article publisher is missing logo"})
		}
	} else if !hasProperty(obj, "publisher") {
		results = append(results, Result{cat, typ + " publisher", SeverityWarning,
			fmt.Sprintf("%s missing publisher (recommended for Article rich result)", typ)})
	}

	if !hasProperty(obj, "dateModified") {
		results = append(results, Result{cat, typ + " dateModified", SeverityInfo,
			fmt.Sprintf("%s missing dateModified (recommended)", typ)})
	}

	if eligible {
		results = append([]Result{{cat, typ + " rich result", SeverityPass,
			"Eligible for Article rich result"}}, results...)
	}

	return results
}

func checkProductRichResult(obj map[string]interface{}) []Result {
	var results []Result
	cat := "Rich Results"

	hasRating := hasProperty(obj, "aggregateRating")
	hasReview := hasProperty(obj, "review")
	hasOffers := hasProperty(obj, "offers")

	if !hasOffers {
		results = append(results, Result{cat, "Product offers", SeverityWarning,
			"Product missing offers (required for price display in SERPs)"})
	} else {
		// Validate offers has price and priceCurrency
		if offers, ok := obj["offers"].(map[string]interface{}); ok {
			if !hasProperty(offers, "price") && !hasProperty(offers, "lowPrice") {
				results = append(results, Result{cat, "Product offers.price", SeverityWarning,
					"Product offers missing price or lowPrice"})
			}
			if !hasProperty(offers, "priceCurrency") {
				results = append(results, Result{cat, "Product offers.priceCurrency", SeverityWarning,
					"Product offers missing priceCurrency"})
			}
		}
	}

	if !hasRating && !hasReview {
		results = append(results, Result{cat, "Product ratings", SeverityInfo,
			"Product has no aggregateRating or review (needed for star ratings in SERPs)"})
	} else if hasRating {
		if rating, ok := obj["aggregateRating"].(map[string]interface{}); ok {
			if !hasProperty(rating, "ratingValue") {
				results = append(results, Result{cat, "Product aggregateRating", SeverityWarning,
					"aggregateRating missing ratingValue"})
			}
			if !hasProperty(rating, "reviewCount") && !hasProperty(rating, "ratingCount") {
				results = append(results, Result{cat, "Product aggregateRating", SeverityWarning,
					"aggregateRating missing reviewCount or ratingCount"})
			}
		}
	}

	if hasProperty(obj, "name") && (hasOffers || hasRating || hasReview) {
		results = append([]Result{{cat, "Product rich result", SeverityPass,
			"Eligible for Product rich result"}}, results...)
	}

	return results
}

func checkBreadcrumbRichResult(obj map[string]interface{}) []Result {
	var results []Result
	cat := "Rich Results"

	items, ok := obj["itemListElement"]
	if !ok {
		return results
	}

	itemList, ok := items.([]interface{})
	if !ok {
		results = append(results, Result{cat, "Breadcrumb items", SeverityError,
			"BreadcrumbList itemListElement should be an array"})
		return results
	}

	validItems := 0
	for i, item := range itemList {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if !hasProperty(entry, "position") {
			results = append(results, Result{cat, fmt.Sprintf("Breadcrumb #%d", i+1), SeverityError,
				"Breadcrumb item missing position"})
		}
		if !hasProperty(entry, "name") && !hasProperty(entry, "item") {
			results = append(results, Result{cat, fmt.Sprintf("Breadcrumb #%d", i+1), SeverityError,
				"Breadcrumb item missing name and item"})
		} else {
			validItems++
		}
	}

	if validItems > 0 {
		results = append([]Result{{cat, "Breadcrumb rich result", SeverityPass,
			fmt.Sprintf("Eligible: %d breadcrumb item(s)", validItems)}}, results...)
	}

	return results
}

func validateTypeSpecific(obj map[string]interface{}, typ string) []Result {
	var results []Result
	cat := "Schema Validation"

	switch typ {
	case "Article", "NewsArticle", "BlogPosting":
		// Validate author is an object with name, not just a string
		if author, ok := obj["author"]; ok {
			switch a := author.(type) {
			case map[string]interface{}:
				if !hasProperty(a, "name") {
					results = append(results, Result{cat, typ + " author.name", SeverityWarning,
						"author object is missing name"})
				}
			case string:
				results = append(results, Result{cat, typ + " author format", SeverityWarning,
					"author should be a Person/Organization object, not a plain string"})
			}
		}

		// Validate publisher has logo
		if publisher, ok := obj["publisher"].(map[string]interface{}); ok {
			if !hasProperty(publisher, "logo") {
				results = append(results, Result{cat, typ + " publisher.logo", SeverityWarning,
					"publisher is missing logo (required by Google for Article)"})
			}
		}

	case "Product":
		// Validate offers has availability
		if offers, ok := obj["offers"].(map[string]interface{}); ok {
			if !hasProperty(offers, "availability") {
				results = append(results, Result{cat, "Product offers.availability", SeverityInfo,
					"Product offers missing availability (recommended)"})
			}
		}

	case "FAQPage":
		// Validate mainEntity contains Questions
		if mainEntity, ok := obj["mainEntity"].([]interface{}); ok {
			for _, item := range mainEntity {
				if q, ok := item.(map[string]interface{}); ok {
					if getType(q) != "Question" {
						results = append(results, Result{cat, "FAQPage mainEntity type", SeverityError,
							fmt.Sprintf("FAQPage mainEntity items should be @type Question, got %q", getType(q))})
						break
					}
				}
			}
		}
	}

	return results
}

// getType extracts @type from a JSON-LD object, handling string and array forms.
func getType(obj map[string]interface{}) string {
	switch t := obj["@type"].(type) {
	case string:
		return t
	case []interface{}:
		if len(t) > 0 {
			if s, ok := t[0].(string); ok {
				return s
			}
		}
	}
	return ""
}

// hasProperty checks if a JSON-LD object has a non-empty property.
func hasProperty(obj map[string]interface{}, key string) bool {
	val, ok := obj[key]
	if !ok {
		return false
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	case nil:
		return false
	default:
		return true
	}
}
