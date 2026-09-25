import "./App.css";
import { Route, Routes } from "react-router";
import HomePage from "./components/Home";
import SignupPage from "./components/user/Signup";
import AuthProvider, { RequireAuth } from "./components/auth/AuthProvider";
import ProfilePage from "./components/user/Profile";
import SigninPage from "./components/user/Signin";
import RecipesPage from "./components/recipe/RecipesPage";
import RecipeDetailPage from "./components/recipe/RecipeDetailPage";
import CreateRecipePage from "./components/recipe/CreateRecipePage";
import EditRecipePage from "./components/recipe/EditRecipePage";
import Navbar from "./components/navigation/Navbar";
import Signout from "./components/user/Signout";

function App() {
  return (
    <AuthProvider>
      <header>
        <Navbar />
      </header>
      <main>
        <Routes>
          <Route index element={<HomePage />} />
          <Route path="users">
            <Route
              index
              element={
                <RequireAuth>
                  <ProfilePage />
                </RequireAuth>
              }
            />
            <Route path="signup" element={<SignupPage />} />
            <Route path="signin" element={<SigninPage />} />
            <Route path="signout" element={<Signout />} />
          </Route>
          <Route path="recipes">
            <Route index element={<RecipesPage />} />
            <Route path=":id" element={<RecipeDetailPage />} />
            <Route
              path="create"
              element={
                <RequireAuth>
                  <CreateRecipePage />
                </RequireAuth>
              }
            />
            <Route
              path=":id/edit"
              element={
                <RequireAuth>
                  <EditRecipePage />
                </RequireAuth>
              }
            />
          </Route>
        </Routes>
      </main>
      <footer></footer>
    </AuthProvider>
  );
}

export default App;
