import { Link, useParams } from "react-router";
import {
  apiClient,
  GetRecipeIngredientResponse,
  GetRecipeResponse,
  GetStepResponse,
} from "../../utils/api";
import { useEffect, useState } from "react";
import RecipeMeta from "./RecipeMeta";

const RecipeDetailPage = () => {
  const { id } = useParams();
  const [recipe, setRecipe] = useState<GetRecipeResponse>();

  useEffect(() => {
    const fetchRecipe = async () => {
      setRecipe(await apiClient.getRecipe(Number(id)));
    };
    fetchRecipe();
  }, [id]);

  return (
    recipe && (
      <>
        <article className="recipe">
          <header className="summary">
            <hgroup className="title">
              <h1>
                {recipe.title} (
                <i>
                  <Link to={`edit`}>Edit</Link>
                </i>
                )
              </h1>
              <p className="description">{recipe.description}</p>
            </hgroup>
            <div className="meta">
              <RecipeMeta recipe={recipe} />
            </div>
          </header>
          <div className="content">
            <section className="ingredients">
              <h2>Ingredients</h2>
              <ul>
                {recipe.ingredients.map(
                  (ingredient: GetRecipeIngredientResponse, i: number) => (
                    <li key={i}>
                      {ingredient.quantity} {ingredient.units}{" "}
                      {ingredient.ingredient}, {ingredient.preparation}
                    </li>
                  ),
                )}
              </ul>
            </section>
            <section className="steps">
              <h2>Steps</h2>
              <ol>
                {recipe.steps
                  .sort((a, b) => a.ordinal - b.ordinal)
                  .map((step: GetStepResponse, i: number) => (
                    <li key={i}>{step.instruction}</li>
                  ))}
              </ol>
            </section>
          </div>
        </article>
      </>
    )
  );
};

export default RecipeDetailPage;
