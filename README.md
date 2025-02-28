# TODO:



- Update front end to have login page first to receive JWT token and hit register + login: DONE
  - Create Login page to hit login route, store jwt token, and move to chat interface page: DONE
  - Create Register page where it hits register and sends them back to login: DONE

- Update front end to hit get chats for a user using JWT token and display the json as button choices in the left sidebar.
  - Hit route and store in frontend: Dpme
  - Display lazily in sidebar with current chat highlighted
  - For simplicity display on the last chat

- Change AddInteraction to now be used by routes and to be internal server function.
  - Remove route for add interaction. : DONE
  - Add chatid to Request struct. : DONE
  - Call through chat.go after a request has been completed by passing chatid. query, model response. : DONE
  - In vLlmInteractor, stuff model response and then call addInteraction function once done.
  - Verify that everything is correct.

- Write new Web Socket Handler that will be unused for now and needs a JWT token to operate.: DONE

- Move my LLM into the vLLM container.
  - Will be slower to load so do this last and continue testing with super lightweight LLM.

- Fix timeouts to something normal
  - In model_service.go and do this last.

- Introduce DDoS safety in server.

- Deploy on Azure
  - Create a single test account and temporarily disable register page.
  - Get Kubernetes configs working. Want to scale vLLM containers mostly.

- Cool features to add eventually
  - Use small LLM to add titles to chats based on queries.
  - Change frontend to actual framework.S