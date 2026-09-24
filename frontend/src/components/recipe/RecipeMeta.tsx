import { GetRecipeResponse } from "../../utils/api";

const RecipeMeta = ({ recipe }: { recipe: GetRecipeResponse }) => {
  return (
    <dl>
      <dt>Author</dt>
      <dd>{recipe.author.name}</dd>
      {recipe.prepTime !== null && (
        <>
          <dt>Prep Time</dt>
          <dd>{recipe.prepTime} seconds</dd>
        </>
      )}
      {recipe.cookTime !== null && (
        <>
          <dt>Cook Time</dt>
          <dd>{recipe.cookTime} seconds</dd>
        </>
      )}
      {recipe.inactiveTime !== null && (
        <>
          <dt>Inactive Time</dt>
          <dd>{recipe.inactiveTime} seconds</dd>
        </>
      )}
      <dt>Yields</dt>
      <dd>
        {recipe.yieldQuantity} {recipe.yieldUnits}
      </dd>
    </dl>
  );
};

export default RecipeMeta;
