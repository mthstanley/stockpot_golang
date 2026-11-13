import { createSearchParams, NavLink, useNavigate } from "react-router";
import { useAuth } from "../auth/authContext";
import { config } from "../../config";
import { ChangeEvent, FormEvent, useState } from "react";

const Navbar = () => {
  const auth = useAuth();

  const [search, setSearch] = useState("");
  const navigate = useNavigate();
  const handleSearch = (e: FormEvent) => {
    e.preventDefault();
    if (search.trim()) {
      navigate({
        pathname: "/recipes",
        search: `?${createSearchParams({ search })}`,
      });
    } else {
      navigate("/recipes");
    }
  };

  return (
    <nav className="navbar">
      <div className="content">
        <div className="brand">
          <h2>Stockpot</h2>
        </div>
        <div className="searchbar">
          <form className="search" onSubmit={handleSearch}>
            <input
              className="search"
              type="text"
              name="search"
              value={search}
              onChange={(e: ChangeEvent<HTMLInputElement>) =>
                setSearch(e.target.value)
              }
              placeholder="What's on the menu?"
            />
          </form>
        </div>
        <div className="links">
          <ul>
            <li>
              <NavLink to="/">Home</NavLink>
            </li>
            <li>
              <NavLink to="recipes">Recipes</NavLink>
            </li>
            {auth.user ? (
              <>
                <li>
                  <NavLink to="users">Profile</NavLink>
                </li>
                <li>
                  <NavLink to="users/signout">Sign-out</NavLink>
                </li>
              </>
            ) : (
              <>
                <li>
                  <NavLink to="users/signin">Sign-in</NavLink>
                </li>
                {!config.disableUserSignup && (
                  <li>
                    <NavLink to="users/signup">Sign-up</NavLink>
                  </li>
                )}
              </>
            )}
          </ul>
        </div>
      </div>
    </nav>
  );
};

export default Navbar;
