import { Link } from "react-router";
import { GetRecipeResponse } from "../../utils/api";
import RecipeMeta from "./RecipeMeta";

type HeadingLevel = "h1" | "h2" | "h3" | "h4" | "h5" | "h6";

const RecipeCard = ({
  recipe,
  headingLevel,
}: {
  recipe: GetRecipeResponse;
  headingLevel: HeadingLevel;
}) => {
  const Heading = headingLevel;
  return (
    <section className="recipe-card">
      <Heading>
        <Link to={`/recipes/${recipe.id}`}>{recipe.title}</Link>
      </Heading>
      <p>
        <RecipeMeta recipe={recipe} />
      </p>
      <p>{recipe.description}</p>
    </section>
  );
};

export default RecipeCard;
