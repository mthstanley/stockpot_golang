package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mthstanley/stockpot/internal/core/recipe"
	"github.com/mthstanley/stockpot/internal/core/user"
)

type Units int

const (
	Grams Units = iota
)

var unitsToString = map[Units]string{
	Grams: "grams",
}

var stringToUnits = map[string]Units{
	"grams": Grams,
}

func (u Units) String() string {
	return unitsToString[u]
}

func (u Units) MarshalJSON() ([]byte, error) {
	str, ok := unitsToString[u]
	if !ok {
		return nil, fmt.Errorf("invalid units value: %d", u)
	}
	return json.Marshal(str)
}

func (u *Units) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	val, ok := stringToUnits[str]
	if !ok {
		return fmt.Errorf("invalid units string: %q", str)
	}

	*u = val
	return nil
}

type JSONSeconds time.Duration

func (j *JSONSeconds) Duration() *time.Duration {
	if j == nil {
		return nil
	}
	td := time.Duration(*j)
	return &td
}

func (j JSONSeconds) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Duration().Seconds())
}

func (j *JSONSeconds) UnmarshalJSON(data []byte) error {
	var seconds int
	if err := json.Unmarshal(data, &seconds); err != nil {
		return err
	}

	val := time.Duration(seconds) * time.Second

	*j = JSONSeconds(val)
	return nil
}

func toJSONSeconds(t *time.Duration) *JSONSeconds {
	if t == nil {
		return nil
	}
	val := JSONSeconds(*t)
	return &val
}

type GetRecipeIngredient struct {
	ID          int64  `json:"id"`
	Ingredient  string `json:"ingredient"`
	Quantity    int    `json:"quantity"`
	Units       string `json:"units"`
	Preparation string `json:"preparation"`
}

func convertToGetRecipeIngredient(r recipe.RecipeIngredient) GetRecipeIngredient {
	var id int64 = -1
	if r.ID != nil {
		id = *r.ID
	}
	return GetRecipeIngredient{
		ID:          id,
		Ingredient:  r.Ingredient.Name,
		Quantity:    r.Quantity,
		Units:       r.Units.Name,
		Preparation: r.Preparation,
	}
}

type GetStep struct {
	ID          int64  `json:"id"`
	Ordinal     int    `json:"ordinal"`
	Instruction string `json:"instruction"`
}

func convertToGetStep(s recipe.Step) GetStep {
	var id int64 = -1
	if s.ID != nil {
		id = *s.ID
	}
	return GetStep{
		ID:          id,
		Ordinal:     s.Ordinal,
		Instruction: s.Instruction,
	}
}

type GetRecipe struct {
	ID            int64                 `json:"id"`
	Title         string                `json:"title"`
	Description   *string               `json:"description"`
	Author        GetUser               `json:"author"`
	PrepTime      *JSONSeconds          `json:"prep_time"`
	CookTime      *JSONSeconds          `json:"cook_time"`
	InactiveTime  *JSONSeconds          `json:"inactive_time"`
	YieldQuantity int                   `json:"yield_quantity"`
	YieldUnits    string                `json:"yield_units"`
	Ingredients   []GetRecipeIngredient `json:"ingredients"`
	Steps         []GetStep             `json:"steps"`
}

func convertToGetRecipe(r recipe.Recipe) GetRecipe {
	var id int64 = -1
	if r.ID != nil {
		id = *r.ID
	}
	var i []GetRecipeIngredient
	for _, ri := range r.Ingredients {
		i = append(i, convertToGetRecipeIngredient(ri))
	}
	var s []GetStep
	for _, rs := range r.Steps {
		s = append(s, convertToGetStep(rs))
	}
	return GetRecipe{
		ID:            id,
		Title:         r.Title,
		Description:   r.Description,
		Author:        convertToGetUser(&r.Author),
		PrepTime:      toJSONSeconds(r.PrepTime),
		CookTime:      toJSONSeconds(r.CookTime),
		InactiveTime:  toJSONSeconds(r.InactiveTime),
		YieldQuantity: r.YieldQuantity,
		YieldUnits:    r.YieldUnits.Name,
		Ingredients:   i,
		Steps:         s,
	}
}

type CreateRecipeIngredient struct {
	Ingredient  string `json:"ingredient"`
	Quantity    int    `json:"quantity"`
	Units       Units  `json:"units"`
	Preparation string `json:"preparation"`
}

func (c CreateRecipeIngredient) convertToRecipeIngredient() recipe.RecipeIngredient {
	return recipe.RecipeIngredient{
		ID:          nil,
		RecipeID:    nil,
		Ingredient:  recipe.Ingredient{ID: nil, Name: c.Ingredient},
		Quantity:    c.Quantity,
		Units:       recipe.Unit{ID: nil, Name: c.Units.String()},
		Preparation: c.Preparation,
	}
}

type CreateStep struct {
	Ordinal     int    `json:"ordinal"`
	Instruction string `json:"instruction"`
}

func (c CreateStep) convertToStep() recipe.Step {
	return recipe.Step{
		ID:          nil,
		RecipeID:    nil,
		Ordinal:     c.Ordinal,
		Instruction: c.Instruction,
	}
}

type CreateRecipe struct {
	Title         string                   `json:"title"`
	Description   *string                  `json:"description"`
	PrepTime      *JSONSeconds             `json:"prep_time"`
	CookTime      *JSONSeconds             `json:"cook_time"`
	InactiveTime  *JSONSeconds             `json:"inactive_time"`
	YieldQuantity int                      `json:"yield_quantity"`
	YieldUnits    Units                    `json:"yield_units"`
	Ingredients   []CreateRecipeIngredient `json:"ingredients"`
	Steps         []CreateStep             `json:"steps"`
}

func (c CreateRecipe) convertToRecipe(u user.User) recipe.Recipe {
	var i []recipe.RecipeIngredient
	for _, ri := range c.Ingredients {
		i = append(i, ri.convertToRecipeIngredient())
	}
	var s []recipe.Step
	for _, rs := range c.Steps {
		s = append(s, rs.convertToStep())
	}
	return recipe.Recipe{
		ID:            nil,
		Title:         c.Title,
		Description:   c.Description,
		Author:        u,
		PrepTime:      c.PrepTime.Duration(),
		CookTime:      c.CookTime.Duration(),
		InactiveTime:  c.InactiveTime.Duration(),
		YieldQuantity: c.YieldQuantity,
		YieldUnits:    recipe.Unit{ID: nil, Name: c.YieldUnits.String()},
		Ingredients:   i,
		Steps:         s,
	}
}

type UpdateRecipeIngredient struct {
	ID          *int64 `json:"id"`
	Ingredient  string `json:"ingredient"`
	Quantity    int    `json:"quantity"`
	Units       Units  `json:"units"`
	Preparation string `json:"preparation"`
}

func (u UpdateRecipeIngredient) convertToRecipeIngredient() recipe.RecipeIngredient {
	return recipe.RecipeIngredient{
		ID:          u.ID,
		RecipeID:    nil,
		Ingredient:  recipe.Ingredient{ID: nil, Name: u.Ingredient},
		Quantity:    u.Quantity,
		Units:       recipe.Unit{ID: nil, Name: u.Units.String()},
		Preparation: u.Preparation,
	}
}

type UpdateStep struct {
	ID          *int64 `json:"id"`
	Ordinal     int    `json:"ordinal"`
	Instruction string `json:"instruction"`
}

func (u UpdateStep) convertToStep() recipe.Step {
	return recipe.Step{
		ID:          u.ID,
		RecipeID:    nil,
		Ordinal:     u.Ordinal,
		Instruction: u.Instruction,
	}
}

type UpdateRecipe struct {
	Title         string                   `json:"title"`
	Description   *string                  `json:"description"`
	PrepTime      *JSONSeconds             `json:"prep_time"`
	CookTime      *JSONSeconds             `json:"cook_time"`
	InactiveTime  *JSONSeconds             `json:"inactive_time"`
	YieldQuantity int                      `json:"yield_quantity"`
	YieldUnits    Units                    `json:"yield_units"`
	Ingredients   []UpdateRecipeIngredient `json:"ingredients"`
	Steps         []UpdateStep             `json:"steps"`
}

func (u UpdateRecipe) convertToRecipe(id int64, author user.User) recipe.Recipe {
	var i []recipe.RecipeIngredient
	for _, ri := range u.Ingredients {
		i = append(i, ri.convertToRecipeIngredient())
	}
	var s []recipe.Step
	for _, rs := range u.Steps {
		s = append(s, rs.convertToStep())
	}
	return recipe.Recipe{
		ID:            &id,
		Title:         u.Title,
		Description:   u.Description,
		Author:        author,
		PrepTime:      u.PrepTime.Duration(),
		CookTime:      u.CookTime.Duration(),
		InactiveTime:  u.InactiveTime.Duration(),
		YieldQuantity: u.YieldQuantity,
		YieldUnits:    recipe.Unit{ID: nil, Name: u.YieldUnits.String()},
		Ingredients:   i,
		Steps:         s,
	}
}

type RecipeHandler struct {
	recipeService recipe.Service
}

func NewRecipeHandler(recipeService recipe.Service) *RecipeHandler {
	return &RecipeHandler{recipeService}
}

func (h RecipeHandler) HandleGetRecipes(w http.ResponseWriter, r *http.Request) error {
	recipes, err := h.recipeService.Get(r.Context())
	if err != nil {
		return fmt.Errorf("failed to get recipe: %w", err)
	}

	getRecipes := []GetRecipe{}
	for _, rec := range recipes {
		getRecipes = append(getRecipes, convertToGetRecipe(rec))
	}
	data, err := json.Marshal(getRecipes)
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}

func (h RecipeHandler) HandleGetRecipe(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return &PathValueParseError{Path: "/recipe/{id}", Err: err}
	}

	rec, err := h.recipeService.GetByID(r.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to get recipe: %w", err)
	}

	data, err := json.Marshal(convertToGetRecipe(*rec))
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}

func (h RecipeHandler) HandleCreateRecipe(w http.ResponseWriter, r *http.Request) error {
	authUser, err := extractAuthUser(r)
	if err != nil {
		return fmt.Errorf("auth user is missing for handler requiring auth: %w", err)
	}

	var createRecipe CreateRecipe
	err = json.NewDecoder(r.Body).Decode(&createRecipe)
	if err != nil {
		return errors.Join(RequestBodyDecodeError, err)
	}

	createdRecipe, err := h.recipeService.Create(r.Context(), createRecipe.convertToRecipe(authUser.User))
	if err != nil {
		return fmt.Errorf("failed to create recipe: %w", err)
	}

	data, err := json.Marshal(convertToGetRecipe(*createdRecipe))
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}

func (h RecipeHandler) HandleUpdateRecipe(w http.ResponseWriter, r *http.Request) error {
	authUser, err := extractAuthUser(r)
	if err != nil {
		return fmt.Errorf("auth user is missing for handler requiring auth: %w", err)
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return &PathValueParseError{Path: "/recipe/{id}", Err: err}
	}

	var updateRecipe UpdateRecipe
	err = json.NewDecoder(r.Body).Decode(&updateRecipe)
	if err != nil {
		return errors.Join(RequestBodyDecodeError, err)
	}

	existingRecipe, err := h.recipeService.GetByID(r.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to fetch recipe for id %d: %w", id, err)
	}

	if !existingRecipe.Author.Equals(authUser.User) {
		return fmt.Errorf("unable to update recipe belonging to another user %s", existingRecipe.Author.Name)
	}

	updatedRecipe, err := h.recipeService.Update(r.Context(), updateRecipe.convertToRecipe(id, authUser.User))
	if err != nil {
		return fmt.Errorf("failed to update recipe: %w", err)
	}

	data, err := json.Marshal(convertToGetRecipe(*updatedRecipe))
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}

func (h RecipeHandler) HandleDeleteRecipe(w http.ResponseWriter, r *http.Request) error {
	authUser, err := extractAuthUser(r)
	if err != nil {
		return fmt.Errorf("auth user is missing for handler requiring auth: %w", err)
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return &PathValueParseError{Path: "/recipe/{id}", Err: err}
	}

	existingRecipe, err := h.recipeService.GetByID(r.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to fetch recipe for id %d: %w", id, err)
	}

	if !existingRecipe.Author.Equals(authUser.User) {
		return fmt.Errorf("unable to delete recipe belonging to another user %s", existingRecipe.Author.Name)
	}

	deletedRecipe, err := h.recipeService.DeleteByID(r.Context(), id)
	if err != nil {
		return fmt.Errorf("failed to delete recipe: %w", err)
	}

	data, err := json.Marshal(convertToGetRecipe(*deletedRecipe))
	if err != nil {
		return fmt.Errorf("failed to json encode response body: %w", err)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("failed to write response body: %w", err)
	}

	return nil
}
