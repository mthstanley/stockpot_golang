import { useFieldArray, useForm } from "react-hook-form";
import {
  apiClient,
  GetRecipeResponse,
  MutateRecipeRequest,
} from "../../utils/api";
import { useNavigate } from "react-router";
import { setEmptyOrStr, stripEmpty } from "../../utils/form";

const RecipeForm = ({ recipe }: { recipe?: MutateRecipeRequest }) => {
  const navigate = useNavigate();
  const { register, control, handleSubmit } = useForm<MutateRecipeRequest>({
    defaultValues: recipe,
  });
  const onSubmit = handleSubmit((recipe: MutateRecipeRequest) => {
    let response: Promise<GetRecipeResponse>;
    const strippedRecipe = stripEmpty(recipe);
    if (recipe.id !== null && recipe.id !== undefined) {
      response = apiClient.updateRecipe(strippedRecipe);
    } else {
      response = apiClient.createRecipe(strippedRecipe);
    }
    response.then((newRecipe) => navigate(`/recipes/${newRecipe.id}`));
  });

  const ingredients = useFieldArray({ control, name: "ingredients" });
  const steps = useFieldArray({ control, name: "steps" });
  const stepIndices: Record<number, number> = steps.fields.reduce(
    (map, { id }, index) => ({ ...map, [id]: index }),
    {},
  );
  const stepOrdinalIndices: Record<number, number> = steps.fields.reduce(
    (map, { ordinal }, index) => ({ ...map, [ordinal]: index }),
    {},
  );
  const sortedStepFields = [...steps.fields].sort(
    (a, b) => a.ordinal - b.ordinal,
  );

  return (
    <form onSubmit={onSubmit} className="recipe">
      <fieldset className="summary">
        <hgroup className="title">
          <h1>
            <input
              id="title"
              placeholder="Title"
              {...register("title", { required: true })}
            />
          </h1>
          <p className="description">
            <textarea
              id="description"
              placeholder="A description of your dish..."
              {...register("description", { setValueAs: setEmptyOrStr })}
            />
          </p>
        </hgroup>
        <div className="meta">
          <dl>
            <dt>
              <label htmlFor="prepTime">Prep Time</label>
            </dt>
            <dd>
              <input
                id="prepTime"
                type="number"
                {...register("prepTime", { valueAsNumber: true })}
              />{" "}
              seconds
            </dd>
            <dt>
              <label htmlFor="cookTime">Cook Time</label>
            </dt>
            <dd>
              <input
                id="cookTime"
                type="number"
                {...register("cookTime", { valueAsNumber: true })}
              />{" "}
              seconds
            </dd>
            <dt>
              <label htmlFor="inactiveTime">Inactive Time</label>
            </dt>
            <dd>
              <input
                id="inactiveTime"
                type="number"
                {...register("inactiveTime", { valueAsNumber: true })}
              />{" "}
              seconds
            </dd>
            <dt>Yields</dt>
            <dd>
              <input
                id="yieldQuantity"
                type="number"
                {...register("yieldQuantity", {
                  required: true,
                  valueAsNumber: true,
                })}
              />
              <select
                id="yieldUnits"
                {...register("yieldUnits", {
                  required: true,
                  setValueAs: setEmptyOrStr,
                })}
              >
                <option value="grams" selected>
                  grams
                </option>
              </select>
            </dd>
          </dl>
        </div>
      </fieldset>
      <div className="content">
        <fieldset className="ingredients">
          <h2>Ingredients</h2>
          <ul>
            {ingredients.fields.map((item, index) => (
              <li key={item.id} className="ingredient">
                <div>
                  <p>
                    <input
                      id={`ingredients.${index}.quantity`}
                      type="number"
                      {...register(`ingredients.${index}.quantity`, {
                        required: true,
                        valueAsNumber: true,
                      })}
                    />
                    <select
                      id={`ingredients.${index}.units`}
                      {...register(`ingredients.${index}.units`, {
                        required: true,
                        setValueAs: setEmptyOrStr,
                      })}
                    >
                      <option value="grams" selected>
                        grams
                      </option>
                    </select>
                    <input
                      id={`ingredients.${index}.ingredient`}
                      placeholder="Ingredient"
                      {...register(`ingredients.${index}.ingredient`, {
                        required: true,
                        setValueAs: setEmptyOrStr,
                      })}
                    />
                    <input
                      id={`ingredients.${index}.preparation`}
                      placeholder="Preparation"
                      {...register(`ingredients.${index}.preparation`, {
                        setValueAs: setEmptyOrStr,
                      })}
                    />
                  </p>
                  <p>
                    <button
                      type="button"
                      onClick={() => ingredients.remove(index)}
                    >
                      Delete
                    </button>
                  </p>
                </div>
              </li>
            ))}
          </ul>
          <button
            type="button"
            onClick={() =>
              ingredients.append({
                ingredient: "",
                quantity: NaN,
                units: "grams",
                preparation: "",
              })
            }
          >
            Add Ingredient
          </button>
        </fieldset>

        <fieldset className="steps">
          <h2>Steps</h2>
          <ol>
            {sortedStepFields.map((item) => (
              <li key={item.id} className="step">
                <textarea
                  id={`steps.${stepIndices[item.id]}.instruction`}
                  placeholder="A description of what to do..."
                  {...register(`steps.${stepIndices[item.id]}.instruction`, {
                    required: true,
                    setValueAs: setEmptyOrStr,
                  })}
                />
              </li>
            ))}
          </ol>
          <button
            type="button"
            onClick={() =>
              steps.append({ ordinal: steps.fields.length, instruction: "" })
            }
          >
            Add Step
          </button>
          {steps.fields.length > 0 && (
            <button
              type="button"
              onClick={() =>
                steps.remove(stepOrdinalIndices[steps.fields.length - 1])
              }
            >
              Remove Step
            </button>
          )}
        </fieldset>
      </div>

      <input type="submit" />
    </form>
  );
};

export default RecipeForm;
