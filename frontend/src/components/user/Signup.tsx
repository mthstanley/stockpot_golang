import { useNavigate } from "react-router";
import { useAuth } from "../auth/authContext";
import { apiClient } from "../../utils/api";
import { useForm } from "react-hook-form";

type FormValues = {
  name: string;
  username: string;
  password: string;
};

const SignupPage = () => {
  const navigate = useNavigate();
  const auth = useAuth();
  const { register, handleSubmit } = useForm<FormValues>();

  const onSubmit = handleSubmit(async (formValues: FormValues) => {
    await apiClient.createUser(formValues);
    auth.signin(formValues.username, formValues.password, () => {
      // Send them back to the page they tried to visit when they were
      // redirected to the login page. Use { replace: true } so we don't create
      // another entry in the history stack for the login page.  This means that
      // when they get to the protected page and click the back button, they
      // won't end up back on the login page, which is also really nice for the
      // user experience.
      navigate("/users");
    });
  });

  return (
    <div className="auth">
      <header className="auth-header page-title">
        <h1>Sign-Up</h1>
      </header>

      <div className="auth-body">
        <form onSubmit={onSubmit}>
          <p>
            <label htmlFor="name">Name</label>
            <input id="name" {...register("name", { required: true })} />
          </p>
          <p>
            <label htmlFor="username">Username</label>
            <input
              id="username"
              {...register("username", { required: true })}
            />
          </p>
          <p>
            <label htmlFor="password">Password</label>
            <input
              type="password"
              id="password"
              {...register("password", { required: true })}
            />
          </p>
          <p>
            <input type="submit" value="Sign-up" />
          </p>
        </form>
      </div>
    </div>
  );
};

export default SignupPage;
