import { useEffect, useState } from "react";
import { apiClient, ApiResponse, GetRecipeResponse } from "../../utils/api";
import RecipeCard from "./RecipeCard";
import { Link, useSearchParams } from "react-router";

const RecipeList = () => {
  const getQueryInt = (key: string, def: number) => {
    const rawValue = searchParams.get(key);
    const value = rawValue ? parseInt(rawValue, 10) : def;
    return isNaN(value) ? def : value;
  };

  const [searchParams, setSearchParams] = useSearchParams();
  const page = getQueryInt("page", 1);
  const size = getQueryInt("size", 10);
  const search = searchParams.get("search");
  const [data, setData] = useState<ApiResponse<GetRecipeResponse[]> | null>(
    null,
  );

  const getUpdatedSearchString = (key: string, value: string) => {
    const currentParams = new URLSearchParams(searchParams);
    currentParams.set(key, value);
    return currentParams.toString();
  };

  useEffect(() => {
    const fetchData = async () => {
      const response = await apiClient.getRecipes({ page, size, search });
      // redirect to the last page if the requested page is past the end
      if (
        response.meta &&
        response.meta.pages > 0 &&
        page > response.meta.pages
      ) {
        setSearchParams(
          (params) => {
            params.set("page", `${response.meta?.pages}`);
            return params;
          },
          { replace: true },
        );
        return;
      }
      setData(response);
    };
    fetchData();
  }, [page, size, search, setSearchParams]);

  return (
    <>
      <ul className="recipe-list">
        {(data?.data ?? []).map((recipe: GetRecipeResponse, i: number) => (
          <li key={i}>
            <RecipeCard recipe={recipe} headingLevel="h2" />
          </li>
        ))}
      </ul>
      {data?.meta && (
        <div className="pagination">
          <span className="meta">
            <em>
              {data.meta.items === 0
                ? 0
                : (data.meta.page - 1) * data.meta.size + 1}{" "}
              - {Math.min(data.meta.page * data.meta.size, data.meta.items)}
            </em>{" "}
            of <em>{data.meta.items}</em> results
          </span>
          <nav>
            <ul>
              <li>
                <Link
                  className={data.meta.page > 1 ? "enabled" : "disabled"}
                  onClick={(e) => {
                    if (data.meta && data.meta.page <= 1) {
                      e.preventDefault();
                    }
                  }}
                  to={{
                    search: getUpdatedSearchString(
                      "page",
                      `${data.meta.page - 1}`,
                    ),
                  }}
                >
                  <span>&lt;</span>
                </Link>
              </li>
              {data.meta.page > 1 && (
                <>
                  <li>
                    <Link
                      to={{
                        search: getUpdatedSearchString("page", "1"),
                      }}
                    >
                      1
                    </Link>
                  </li>
                  <li className="ellipsis">...</li>
                </>
              )}
              <li>
                <Link
                  className="current-page"
                  to={{
                    search: getUpdatedSearchString("page", `${data.meta.page}`),
                  }}
                >
                  {data.meta.page}
                </Link>
              </li>
              {data.meta.page < data.meta.pages && (
                <>
                  <li className="ellipsis">...</li>
                  <li>
                    <Link
                      to={{
                        search: getUpdatedSearchString(
                          "page",
                          `${data.meta.pages}`,
                        ),
                      }}
                    >
                      {data.meta.pages}
                    </Link>
                  </li>
                </>
              )}
              <li>
                <Link
                  className={
                    data.meta.page < data.meta.pages ? "enabled" : "disabled"
                  }
                  onClick={(e) => {
                    if (data.meta && data.meta.page >= data.meta.pages) {
                      e.preventDefault();
                    }
                  }}
                  to={{
                    search: getUpdatedSearchString(
                      "page",
                      `${data.meta.page + 1}`,
                    ),
                  }}
                >
                  <span>&gt;</span>
                </Link>
              </li>
            </ul>
          </nav>
        </div>
      )}
    </>
  );
};

export default RecipeList;
