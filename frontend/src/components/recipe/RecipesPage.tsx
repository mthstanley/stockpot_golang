import { Link } from "react-router";
import RecipeList from "./RecipeList";

const RecipesPage = () => {
  return (
    <>
      <header className="page-title">
        <h1>
          Recipes (
          <i>
            <Link to="create">New</Link>
          </i>
          )
        </h1>
      </header>
      <RecipeList />
    </>
  );
};

export default RecipesPage;
