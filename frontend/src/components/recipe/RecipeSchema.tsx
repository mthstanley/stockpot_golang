import { Recipe } from "schema-dts";
import { JsonLd } from "react-schemaorg";
import { GetRecipeResponse } from "../../utils/api";

const toIso8601Duration = (seconds: number | undefined) => {
  if (seconds === undefined || seconds === null) {
    return "";
  }
  return `PT${seconds}S`;
};

const totalTime = (recipe: GetRecipeResponse) => {
  const times = [recipe.cookTime, recipe.prepTime, recipe.inactiveTime];
  if (times.every((value) => value === null || value == undefined)) {
    return undefined;
  }
  return times.reduce((prev, current) => (prev ?? 0) + (current ?? 0), 0);
};

const RecipeSchema = ({ recipe }: { recipe: GetRecipeResponse }) => {
  return (
    <JsonLd<Recipe>
      item={{
        "@context": "https://schema.org",
        "@type": "Recipe",
        name: recipe.title,
        author: {
          "@type": "Person",
          name: recipe.author.name,
        },
        prepTime: toIso8601Duration(recipe.prepTime),
        cookTime: toIso8601Duration(recipe.cookTime),
        totalTime: toIso8601Duration(totalTime(recipe)),
        recipeIngredient: recipe.ingredients.map(
          (ingredient) =>
            `${ingredient.quantity} ${ingredient.units} ${ingredient.ingredient}, ${ingredient.preparation}`,
        ),
        recipeInstructions: recipe.steps.map((step) => ({
          "@type": "HowToStep",
          position: step.ordinal,
          text: step.instruction,
        })),
      }}
    />
  );
};

export default RecipeSchema;
