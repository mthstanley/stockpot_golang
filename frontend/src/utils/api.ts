import axios, {
  AxiosError,
  AxiosInstance,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from "axios";
import { config } from "../config";
import { camelKeys, snakeKeys } from "./case";

export interface GetUserResponse {
  id: number;
  name: string;
}

export interface CreateUserRequest {
  username: string;
  password: string;
  name: string;
}

export interface GetTokenResponse {
  token: string;
}

export interface GetRecipeIngredientResponse {
  id: number;
  ingredient: string;
  quantity: number;
  units: string;
  preparation: string;
}

export interface GetStepResponse {
  id: number;
  ordinal: number;
  instruction: string;
}

export interface GetRecipeResponse {
  id: number;
  title: string;
  description?: string;
  author: GetUserResponse;
  prepTime?: number;
  cookTime?: number;
  inactiveTime?: number;
  yieldQuantity: number;
  yieldUnits: string;
  ingredients: Array<GetRecipeIngredientResponse>;
  steps: Array<GetStepResponse>;
}

export interface MutateRecipeIngredientRequest {
  id?: number;
  ingredient: string;
  quantity: number;
  units: string;
  preparation: string;
}

export interface MutateStepRequest {
  id?: number;
  ordinal: number;
  instruction: string;
}

export interface MutateRecipeRequest {
  id?: number;
  title: string;
  description?: string;
  prepTime?: number;
  cookTime?: number;
  inactiveTime?: number;
  yieldQuantity: number;
  yieldUnits: string;
  ingredients: Array<MutateRecipeIngredientRequest>;
  steps: Array<MutateStepRequest>;
}

export interface BasicAuth {
  username: string;
  password: string;
  kind: "basic";
}

export interface BearerAuth {
  token: string;
  kind: "bearer";
}

export type Credentials = BasicAuth | BearerAuth;

class ApiClient {
  client: AxiosInstance;
  tokenExpirationCallback: VoidFunction;

  constructor(urlBase: URL) {
    this.client = axios.create({ baseURL: urlBase.toString() });
    this.client.defaults.headers.post["Content-Type"] = "application/json";
    this.tokenExpirationCallback = () => {};
    this.client.interceptors.response.use(
      (response: AxiosResponse): AxiosResponse => {
        if (
          response.data &&
          response.headers["content-type"].startsWith("application/json")
        ) {
          response.data = camelKeys(response.data);
        }

        return response;
      },
      (error: AxiosError): Promise<AxiosError> => {
        if (error.response) {
          if (error.response.status === 401) {
            this.tokenExpirationCallback();
          }
        }
        return Promise.reject(error);
      },
    );
    this.client.interceptors.request.use(
      (config: InternalAxiosRequestConfig): InternalAxiosRequestConfig => {
        const newConfig: InternalAxiosRequestConfig = { ...config };

        if (newConfig.headers["Content-Type"] === "multipart/form-data") {
          return newConfig;
        }

        if (config.params) {
          newConfig.params = snakeKeys(config.params);
        }

        if (config.data) {
          newConfig.data = snakeKeys(config.data);
        }

        return newConfig;
      },
    );
  }

  async createUser(request: CreateUserRequest): Promise<GetUserResponse> {
    return this.client
      .post<GetUserResponse>("/user", request)
      .then((response) => response.data);
  }

  async login(
    credentials: Credentials,
    tokenExpirationCallback: VoidFunction,
  ): Promise<GetTokenResponse> {
    switch (credentials.kind) {
      case "basic":
        return this.client
          .post<GetTokenResponse>(
            "/user/token",
            {},
            {
              auth: {
                username: credentials.username,
                password: credentials.password,
              },
            },
          )
          .then((response) => {
            this.client.defaults.headers.common["Authorization"] =
              `Bearer ${response.data.token}`;
            this.tokenExpirationCallback = tokenExpirationCallback;
            return response.data;
          });
      case "bearer":
        this.client.defaults.headers.common["Authorization"] =
          `Bearer ${credentials.token}`;
        this.tokenExpirationCallback = tokenExpirationCallback;
        return Promise.resolve({ token: credentials.token });
    }
  }

  logout() {
    this.client.defaults.headers.common["Authorization"] = "";
  }

  async getRecipes(): Promise<Set<GetRecipeResponse>> {
    return this.client
      .get<Set<GetRecipeResponse>>("/recipe")
      .then((response) => response.data);
  }

  async getRecipe(id: number): Promise<GetRecipeResponse> {
    return this.client
      .get<GetRecipeResponse>(`/recipe/${id}`)
      .then((response) => response.data);
  }

  async createRecipe(
    createRecipeRequest: MutateRecipeRequest,
  ): Promise<GetRecipeResponse> {
    return this.client
      .post<GetRecipeResponse>("/recipe", createRecipeRequest)
      .then((response) => response.data);
  }

  async updateRecipe(
    updateRecipeRequest: MutateRecipeRequest,
  ): Promise<GetRecipeResponse> {
    return this.client
      .post(`/recipe/${updateRecipeRequest.id!}`, updateRecipeRequest)
      .then((response) => response.data);
  }
}

export const apiClient = new ApiClient(config.apiBaseUrl);
